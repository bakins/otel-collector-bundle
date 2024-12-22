package victrondbusreceiver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	dbus "github.com/godbus/dbus/v5"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

var typeStr = component.MustNewType("victrondbus")

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	Address                        string `mapstructure:"address"`
}

func (cfg *Config) Validate() error {
	return nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha))
}

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = time.Minute

	return &Config{
		ControllerConfig: cfg,
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	ns := newDbusScraper(params, cfg)
	scraper, err := scraperhelper.NewScraperWithoutType(
		ns.scrape,
		scraperhelper.WithStart(ns.start),
		scraperhelper.WithShutdown(ns.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ControllerConfig, params, consumer,
		scraperhelper.AddScraperWithType(typeStr, scraper),
	)
}

type dbusScraper struct {
	settings component.TelemetrySettings
	cfg      *Config
	conn     *dbus.Conn
}

func newDbusScraper(settings receiver.Settings, cfg *Config) *dbusScraper {
	s := &dbusScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
	}

	return s
}

func (s *dbusScraper) shutdown(_ context.Context) error {
	if s.conn != nil {
		return s.conn.Close()
	}

	return nil
}

func (s *dbusScraper) start(ctx context.Context, host component.Host) error {
	var c *dbus.Conn
	var err error
	if s.cfg.Address != "" {
		c, err = dbus.Connect(s.cfg.Address, dbus.WithAuth(dbus.AuthAnonymous()))
		if err != nil {
			return err
		}
	} else {
		c, err = dbus.ConnectSystemBus()
		if err != nil {
			return err
		}

	}
	s.conn = c

	return nil
}

const victronPrefix = "com.victronenergy."

func (s *dbusScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	if s.conn == nil {
		return metrics, errors.New("dbus connection not configured")
	}

	var names []string
	if err := s.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		s.settings.Logger.Error("failed to list dbus names", zap.Error(err))
		return pmetric.Metrics{}, nil
	}

	for _, name := range names {
		if !strings.HasPrefix(name, victronPrefix) {
			continue
		}

		m, err := s.handleDevice(context.Background(), name)
		if err != nil {
			s.settings.Logger.Error("failed to handle dbus service", zap.String("service", name), zap.Error(err))
		}

		for i := 0; i < m.ResourceMetrics().Len(); i++ {
			m.ResourceMetrics().At(i).MoveTo(metrics.ResourceMetrics().AppendEmpty())
		}
	}

	return metrics, nil
}

func (s *dbusScraper) handleDevice(ctx context.Context, path string) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()

	name := strings.TrimPrefix(path, victronPrefix)
	parts := strings.Split(name, ".")

	deviceType := strings.ToLower(parts[0])

	handler, ok := metricTypes[deviceType]
	if !ok {
		return metrics, nil
	}

	obj := s.conn.Object(path, dbus.ObjectPath("/"))

	values := map[string]map[string]dbus.Variant{}
	if err := obj.CallWithContext(ctx, "GetItems", 0).Store(&values); err != nil {
		return metrics, fmt.Errorf("dbus call GetItems failed for %q %w", path, err)
	}

	converted := convertValues(values)

	r := metrics.ResourceMetrics().AppendEmpty()
	attributes := r.Resource().Attributes()

	instanceID := getDeviceId(converted)

	if deviceType == "system" {
		instanceID = 0
	}

	if instanceID >= 0 {
		attributes.PutStr("instance_id", strconv.Itoa(int(instanceID)))
	}

	attributes.PutStr("component_type", deviceType)

	if customName := getCustomName(converted); customName != "" {
		attributes.PutStr("custom_name", customName)
	}

	mm := r.ScopeMetrics().
		AppendEmpty().
		Metrics()

	// this is not great - could (should?) create a cache
	// of path to handler?
	for _, h := range handler {
		for k, v := range converted {
			if !h.matcher(k) {
				continue
			}
			metric := h.handler(k, v)
			metric.MoveTo(mm.AppendEmpty())
		}
	}

	return metrics, nil
}

// see https://dbus.freedesktop.org/doc/dbus-specification.html
// basic types
// and https://github.com/godbus/dbus/blob/master/sig.go#L9
func convertValues(input map[string]map[string]dbus.Variant) map[string]any {
	output := make(map[string]any, len(input))
	for k, v := range input {
		val, ok := v["Value"]
		if !ok {
			continue
		}

		k = strings.TrimPrefix(k, "/")
		switch vv := val.Value().(type) {
		case int16:
			output[k] = int64(vv)
		case uint16:
			output[k] = int64(vv)
		case int32:
			output[k] = int64(vv)
		case uint32:
			output[k] = int64(vv)
		case int:
			output[k] = int64(vv)
		case uint:
			output[k] = int64(vv)
		case int64:
			output[k] = int64(vv)
		case float64:
			output[k] = toFixed(vv, 3)
		case string:
			output[k] = vv
		}

	}

	return output
}

func getCustomName(values map[string]any) string {
	var name string
	if err := getValue(values, "CustomName", &name); err != nil {
		return ""
	}

	return name
}

func getDeviceId(values map[string]any) int64 {
	var instanceId int64
	if err := getValue(values, "DeviceInstance", &instanceId); err != nil {
		return -1
	}

	return instanceId
}

// see https://github.com/victronenergy/venus/wiki/dbus
// limited to what I care about right now.

// currently only handles inverters
func (s *dbusScraper) handleVebus(remainder string, values map[string]any) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()

	var powerOut float64
	if err := getValue(values, "Ac/Out/P", &powerOut); err != nil {
		return metrics, err
	}

	r := metrics.ResourceMetrics().AppendEmpty()

	attributes := r.Resource().Attributes()
	attributes.PutStr("local_id", remainder)

	m := r.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetDescription("AC power out")
	m.SetName("victron.ac.output.power_watts")
	m.SetEmptyGauge().
		DataPoints().
		AppendEmpty().
		SetDoubleValue(powerOut)

	return metrics, nil
}

var errNotFound = errors.New("value not found")

func getValue(values map[string]any, key string, dest any) error {
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Pointer {
		return fmt.Errorf("must pass pointer, got %T", dest)
	}

	rv := reflect.ValueOf(dest).Elem()

	val, ok := values[key]
	if !ok {
		return errNotFound
	}

	switch rv.Kind() {
	case reflect.Int64:
		var vv int64

		switch actual := val.(type) {
		case float64:
			vv = int64(actual)
		case int64:
			vv = actual
		default:
			return fmt.Errorf("unexpected type %T for key %s", val, key)
		}

		rv.Set(reflect.ValueOf(vv))

		return nil
	case reflect.Float64:
		var vv float64

		switch actual := val.(type) {
		case float64:
			vv = actual
		case int64:
			vv = float64(actual)
		default:
			return fmt.Errorf("unexpected type %T for key %s", val, key)
		}

		rv.Set(reflect.ValueOf(vv))

		return nil

	case reflect.String:
		v, ok := val.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for key %s", val, key)
		}

		rv.Set(reflect.ValueOf(v))

		return nil

	default:
		return fmt.Errorf("unsupported value type %s for key %s", rv.Kind().String(), key)
	}
}

var metricTypes = map[string][]metricHandler{
	"temperature": {
		// https://github.com/victronenergy/venus/wiki/dbus#temperatures
		{
			matcher: exactMatcher("Temperature"),
			handler: gaugeHandler("victron_environment_temperature"),
		},
		{
			matcher: exactMatcher("Humidity"),
			handler: gaugeHandler("victron_environment_humidity"),
		},
	},
	"vebus": {
		{
			handler: gaugeHandler("victron.ac.output_power"),
			matcher: exactMatcher("Ac/Out/P"),
		},
		{
			handler: vebusDeviceHandler(`victron.ac.device.output_power`),
			matcher: regexMatcher(`^Devices/\d+/Ac/Out/P$`),
		},
		{
			handler: vebusDeviceHandler(`victron.ac.device.input_power`),
			matcher: regexMatcher(`^Devices/\d+/Ac/In/P$`),
		},
	},
	"system": {
		{
			matcher: exactMatcher("Ac/Consumption/L1/Power"),
			handler: gaugeHandler("victron.system.ac.consumption.power"),
		},
		{
			matcher: exactMatcher("Ac/Consumption/L1/Current"),
			handler: gaugeHandler("victron.system.ac.consumption.current"),
		},
		{
			matcher: exactMatcher("Ac/Grid/L1/Power"),
			handler: gaugeHandler("victron.system.ac.grid.power"),
		},
		{
			matcher: exactMatcher("Ac/Grid/L1/Current"),
			handler: gaugeHandler("victron.system.ac.grid.current"),
		},
		{
			matcher: exactMatcher("Ac/Genset/L1/Power"),
			handler: gaugeHandler("victron.system.ac.genset.power"),
		},
		{
			matcher: exactMatcher("Ac/Genset/L1/Current"),
			handler: gaugeHandler("victron.system.ac.genset.current"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Power"),
			handler: gaugeHandler("victron.system.battery.power"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Current"),
			handler: gaugeHandler("victron.system.battery.current"),
		},
		{
			matcher: exactMatcher("Dc/Battery/TimeToGo"),
			handler: gaugeHandler("victron.system.battery.timetogo"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Soc"),
			handler: gaugeHandler("victron.system.battery.soc"),
		},
		{
			matcher: exactMatcher("Dc/Pv/Power"),
			handler: gaugeHandler("victron.system.solar.power"),
		},
		{
			matcher: exactMatcher("Dc/Pv/Current"),
			handler: gaugeHandler("victron.system.solar.current"),
		},
		{
			matcher: exactMatcher("Dc/InverterCharger/Power"),
			handler: gaugeHandler("victron.system.invertercharger.power"),
		},
		{
			matcher: exactMatcher("Dc/InverterCharger/Current"),
			handler: gaugeHandler("victron.system.invertercharger.current"),
		},
		{
			matcher: exactMatcher("Dc/System/Power"),
			handler: gaugeHandler("victron.system.dc.power"),
		},
		{
			matcher: exactMatcher("Dc/System/Current"),
			handler: gaugeHandler("victron.system.dc.current"),
		},
		{
			matcher: exactMatcher("Dc/Charger/Power"),
			handler: gaugeHandler("victron.system.charger.power"),
		},
		{
			matcher: exactMatcher("Dc/Charger/Current"),
			handler: gaugeHandler("victron.system.charger.current"),
		},
	},
	"solarcharger": {
		{
			matcher: exactMatcher("Yield/Power"),
			handler: gaugeHandler("victron.solarcharger.yield.power"),
		},
		{
			matcher: exactMatcher("Dc/0/Current"),
			handler: gaugeHandler("victron.solarcharger.dc.current"),
		},
	},
	"tank": {
		{
			matcher: exactMatcher("Level"),
			handler: gaugeHandler("victron.tank.level"),
		},
	},
}

func gaugeHandler(name string) valueHandler {
	return func(path string, value any) pmetric.Metric {
		m := pmetric.NewMetric()
		m.SetName(name)
		m.SetDescription("victron " + name)

		datapoint := m.SetEmptyGauge().
			DataPoints().
			AppendEmpty()

		switch vv := value.(type) {
		case float64:
			datapoint.SetDoubleValue(vv)
		case int64:
			datapoint.SetIntValue(vv)
		}

		return m
	}
}

type metricHandler struct {
	handler valueHandler
	matcher pathMatcher
}

type valueHandler func(path string, value any) pmetric.Metric

type pathMatcher func(string) bool

func exactMatcher(key string) pathMatcher {
	return func(path string) bool {
		return key == path
	}
}

func regexMatcher(pattern string) pathMatcher {
	r := regexp.MustCompile(pattern)

	return r.MatchString
}

func vebusDeviceHandler(name string) func(path string, value any) pmetric.Metric {
	return func(path string, value any) pmetric.Metric {
		// for example: "Devices/0/Ac/Out"
		parts := strings.Split(path, "/")

		var deviceID string
		if len(parts) >= 2 {
			deviceID = parts[1]
		}

		v := value.(int64)

		m := pmetric.NewMetric()
		m.SetName(name)
		m.SetDescription("victron " + name)

		datapoint := m.SetEmptyGauge().
			DataPoints().
			AppendEmpty()

		datapoint.Attributes().PutStr("device_id", deviceID)

		datapoint.SetIntValue(v)

		return m
	}
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

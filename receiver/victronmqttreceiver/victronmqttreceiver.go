package victronmqttreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var typeStr = component.MustNewType("victronmqtt")

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Endpoint string `mapstructure:"endpoint"`
	ClientID string `mapstructure:"client-id"`
}

func (cfg *Config) Validate() error {
	// Validate that the port is in the address
	_, port, err := net.SplitHostPort(cfg.Endpoint)
	if err != nil {
		return err
	}

	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf(`invalid port "%s"`, port)
	}

	return nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha))
}

func createDefaultConfig() component.Config {
	return &Config{
		ClientID: typeStr.String(),
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	m := mqttReceiver{
		config:       *cfg,
		settings:     params.TelemetrySettings,
		nextConsumer: consumer,
	}

	return &m, nil
}

type mqttReceiver struct {
	config         Config
	client         mqtt.Client
	cancelFunc     context.CancelFunc
	settings       component.TelemetrySettings
	systemSerialID string
	errorGroup     *errgroup.Group
	nextConsumer   consumer.Metrics
}

func (m *mqttReceiver) Shutdown(ctx context.Context) error {
	if m.client != nil {
		m.client.Disconnect(1000)
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.errorGroup != nil {
		return m.errorGroup.Wait()
	}

	return nil
}

func (m *mqttReceiver) Start(_ context.Context, _ component.Host) error {
	opts := (&mqtt.ClientOptions{}).
		SetAutoReconnect(true).
		SetKeepAlive(time.Second * 30).
		SetConnectRetry(true).
		SetAutoReconnect(true).
		SetCleanSession(false).
		AddBroker(m.config.Endpoint).
		SetClientID(m.config.ClientID)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.client = client

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		m.publishKeepalive(ctx)
		return nil
	})

	eg.Go(func() error {
		token := m.client.Subscribe("#", 0, func(_ mqtt.Client, msg mqtt.Message) {
			m.handleMessage(msg)
		})

		token.Wait()

		if err := token.Error(); err != nil {
			m.settings.Logger.Warn("failed to subscribe to topic", zap.Error(err))
			return err
		}

		return nil
	})

	m.errorGroup = eg

	return nil
}

func (m *mqttReceiver) publishKeepalive(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check whether we've heard back from victron mqtt yet...
			if m.systemSerialID == "" {
				continue
			}

			token := m.client.Publish(
				fmt.Sprintf("R/%s/system/0/Serial", m.systemSerialID),
				0,
				false,
				[]byte(""),
			)

			token.Wait()

			if err := token.Error(); err != nil {
				m.settings.Logger.Warn("failed to publish keepalive", zap.Error(err))
				continue
			}

			for !token.WaitTimeout(time.Second * 5) {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}

			if err := token.Error(); err != nil {
				m.settings.Logger.Warn("failed to publish keepalive", zap.Error(err))
				continue
			}
		}
	}
}

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

type metricHandler struct {
	parser  func([]byte) (any, error)
	handler mqttHandler
	matcher pathMatcher
}

type mqttHandler func(path string, value any) pmetric.Metric

var topicTypes = map[string][]metricHandler{
	"vebus": {
		{
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.ac.output_power"),
			matcher: exactMatcher("Ac/Out/P"),
		},
		{
			parser:  parseDoubleValue,
			handler: vebusDeviceHandler(`victron.ac.device.output_power`),
			matcher: regexMatcher(`^Devices/\d+/Ac/Out/P$`),
		},
		{
			parser:  parseDoubleValue,
			handler: vebusDeviceHandler(`victron.ac.device.input_power`),
			matcher: regexMatcher(`^Devices/\d+/Ac/In/P$`),
		},
	},
	"system": {
		{
			matcher: exactMatcher("Ac/Consumption/L1/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.consumption.power"),
		},
		{
			matcher: exactMatcher("Ac/Consumption/L1/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.consumption.current"),
		},
		{
			matcher: exactMatcher("Ac/Grid/L1/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.grid.power"),
		},
		{
			matcher: exactMatcher("Ac/Grid/L1/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.grid.current"),
		},
		{
			matcher: exactMatcher("Ac/Genset/L1/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.genset.power"),
		},
		{
			matcher: exactMatcher("Ac/Genset/L1/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.genset.current"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.power"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.current"),
		},
		{
			matcher: exactMatcher("Dc/Battery/TimeToGo"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.timetogo"),
		},
		{
			matcher: exactMatcher("Dc/Battery/Soc"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.soc"),
		},
		{
			matcher: exactMatcher("Dc/Pv/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.solar.power"),
		},
		{
			matcher: exactMatcher("Dc/Pv/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.solar.current"),
		},
		{
			matcher: exactMatcher("Dc/InverterCharger/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.invertercharger.power"),
		},
		{
			matcher: exactMatcher("Dc/InverterCharger/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.invertercharger.current"),
		},
		{
			matcher: exactMatcher("Dc/System/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.dc.power"),
		},
		{
			matcher: exactMatcher("Dc/System/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.dc.current"),
		},
		{
			matcher: exactMatcher("Dc/Charger/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.charger.power"),
		},
		{
			matcher: exactMatcher("Dc/Charger/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.charger.current"),
		},
	},
	"solarcharger": {
		{
			matcher: exactMatcher("Yield/Power"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.solarcharger.yield.power"),
		},
		{
			matcher: exactMatcher("Dc/0/Current"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.solarcharger.dc.current"),
		},
	},
	"tank": {
		{
			matcher: exactMatcher("Level"),
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.tank.level"),
		},
	},
}

func gaugeHandler(name string) mqttHandler {
	return func(path string, value any) pmetric.Metric {
		v := value.(float64)

		m := pmetric.NewMetric()
		m.SetName(name)
		m.SetDescription("victron " + name)

		m.SetEmptyGauge().
			DataPoints().
			AppendEmpty().
			SetDoubleValue(v)

		return m
	}
}

func vebusDeviceHandler(name string) func(path string, value any) pmetric.Metric {
	return func(path string, value any) pmetric.Metric {
		// for example: "Devices/0/Ac/Out"
		parts := strings.Split(path, "/")

		var deviceID string
		if len(parts) >= 2 {
			deviceID = parts[1]
		}

		v := value.(float64)

		m := pmetric.NewMetric()
		m.SetName(name)
		m.SetDescription("victron " + name)

		datapoint := m.SetEmptyGauge().
			DataPoints().
			AppendEmpty()

		datapoint.Attributes().PutStr("device_id", deviceID)

		datapoint.SetDoubleValue(v)

		return m
	}
}

type victronValue struct {
	Value *float64 `json:"value"`
}

func parseDoubleValue(payload []byte) (any, error) {
	var v victronValue

	fmt.Println("parseDoubleValue", string(payload))

	err := json.Unmarshal(payload, &v)
	if err != nil {
		return nil, err
	}

	if v.Value == nil {
		val := math.NaN()
		v.Value = &val
	}

	return *v.Value, nil
}

func (m *mqttReceiver) handleMessage(msg mqtt.Message) {
	topic := msg.Topic()
	topicParts := strings.Split(topic, "/")

	if len(topicParts) < 5 {
		return
	}

	topicInfoParts := topicParts[4:]
	componentType := topicParts[2]
	componentID := topicParts[3]

	topicString := strings.Join(topicInfoParts, "/")

	if (topicString == "Serial") && (m.systemSerialID == "") {
		value, err := parseStringValue(msg.Payload())
		if err != nil {
			return
		}

		if value == "" {
			return
		}

		// technically a race, but the pattern of publishes makes it "okay"
		m.systemSerialID = value.(string)

		return
	}

	handlers, ok := topicTypes[componentType]
	if !ok {
		return
	}

	var handler metricHandler

	for _, h := range handlers {
		if h.matcher != nil && h.matcher(topicString) {
			handler = h
			break
		}
	}

	if handler.parser == nil || handler.handler == nil {
		return
	}

	value, err := handler.parser(msg.Payload())
	if err != nil {
		m.settings.Logger.Warn(
			"failed to unmarshal payload",
			zap.String("topic", topic),
			zap.String("payload", string(msg.Payload())),
			zap.Error(err),
		)
	}

	metrics := pmetric.NewMetrics()
	r := metrics.ResourceMetrics().AppendEmpty()
	attributes := r.Resource().Attributes()
	attributes.PutStr("instance_id", componentID)
	attributes.PutStr("component_type", componentType)

	mm := r.ScopeMetrics().
		AppendEmpty().
		Metrics()

	metric := handler.handler(topicString, value)
	metric.MoveTo(mm.AppendEmpty())

	if err := m.nextConsumer.ConsumeMetrics(context.Background(), metrics); err != nil {
		m.settings.Logger.Warn("failed to pass metrics to next consumer", zap.Error(err))
	}
}

func parseStringValue(payload []byte) (any, error) {
	var v victronStringValue

	err := json.Unmarshal(payload, &v)
	if err != nil {
		return nil, err
	}
	if v.Value != nil {
		return *v.Value, nil
	}
	return "", nil
}

type victronStringValue struct {
	Value *string `json:"value"`
}

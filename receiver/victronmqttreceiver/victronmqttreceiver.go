package victronmqttreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"gobot.io/x/gobot/v2/platforms/mqtt"
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

	adapter := mqtt.NewAdaptor(cfg.Endpoint, cfg.ClientID)

	m := mqttReceiver{
		adapter:      adapter,
		settings:     params.TelemetrySettings,
		nextConsumer: consumer,
	}

	return &m, nil
}

type mqttReceiver struct {
	adapter        *mqtt.Adaptor
	cancelFunc     context.CancelFunc
	settings       component.TelemetrySettings
	systemSerialID string
	errorGroup     *errgroup.Group
	nextConsumer   consumer.Metrics
}

func (m *mqttReceiver) Shutdown(ctx context.Context) error {
	if m.adapter != nil {
		// this never returns an error
		_ = m.adapter.Disconnect()
		m.adapter = nil
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
	m.adapter.SetAutoReconnect(true)
	m.adapter.SetCleanSession(false)

	if err := m.adapter.Connect(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		m.publishKeepalive(ctx)
		return nil
	})

	eg.Go(func() error {
		token, err := m.adapter.OnWithQOS("#", 0, m.handleMessage)
		if err != nil {
			m.settings.Logger.Error("failed to subscribe to topic", zap.Error(err))
			return err
		}

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

			token, err := m.adapter.PublishWithQOS(
				fmt.Sprintf("R/%s/system/0/Serial", m.systemSerialID),
				0,
				[]byte(""),
			)
			if err != nil {
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

type metricHandler struct {
	parser  func([]byte) (any, error)
	handler func(any) pmetric.Metric
}

var topicTypes = map[string]map[string]metricHandler{
	"vebus": {
		"Ac/Out/P": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.ac.output_power"),
		},
	},
	"system": {
		"Ac/Consumption/L1/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.consumption.power"),
		},
		"Ac/Consumption/L1/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.consumption.current"),
		},
		"Ac/Grid/L1/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.grid.power"),
		},
		"Ac/Grid/L1/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.grid.current"),
		},
		"Ac/Genset/L1/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.genset.power"),
		},
		"Ac/Genset/L1/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.ac.genset.current"),
		},
		"Dc/Battery/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.power"),
		},
		"Dc/Battery/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.current"),
		},
		"Dc/Battery/TimeToGo": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.timetogo"),
		},
		"Dc/Battery/Soc": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.battery.soc"),
		},
		"Dc/Pv/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.solar.power"),
		},
		"Dc/Pv/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.solar.current"),
		},
		"Dc/InverterCharger/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.invertercharger.power"),
		},
		"Dc/InverterCharger/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.invertercharger.current"),
		},
		"Dc/System/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.dc.power"),
		},
		"Dc/System/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.dc.current"),
		},
		"Dc/Charger/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.charger.power"),
		},
		"Dc/Charger/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.system.charger.current"),
		},
	},
	"solarcharger": {
		"Yield/Power": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.solarcharger.yield.power"),
		},
		"Dc/0/Current": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.solarcharger.dc.current"),
		},
	},
	"tank": {
		"Level": {
			parser:  parseDoubleValue,
			handler: gaugeHandler("victron.tank.level"),
		},
	},
}

func gaugeHandler(name string) func(any) pmetric.Metric {
	return func(value any) pmetric.Metric {
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

type victronValue struct {
	Value *float64 `json:"value"`
}

func parseDoubleValue(payload []byte) (any, error) {
	var v victronValue

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

	handler, ok := handlers[topicString]
	if !ok {
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

	metric := handler.handler(value)
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

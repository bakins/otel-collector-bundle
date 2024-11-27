package mqttexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"gobot.io/x/gobot/v2/platforms/mqtt"
)

var typeStr = component.MustNewType("mqtt")

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

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ClientID: typeStr.String(),
	}
}

func createMetricsExporter(
	ctx context.Context,
	settings exporter.Settings,
	rConf component.Config,
) (exporter.Metrics, error) {
	cfg := rConf.(*Config)

	adapter := mqtt.NewAdaptor(cfg.Endpoint, cfg.ClientID)

	m := mqttExporter{
		adapter:  adapter,
		settings: settings.TelemetrySettings,
	}

	return exporterhelper.NewMetrics(
		ctx,
		settings,
		cfg,
		m.consumeMetrics,
		exporterhelper.WithStart(m.Start),
		exporterhelper.WithShutdown(m.Shutdown),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
}

type mqttExporter struct {
	adapter      *mqtt.Adaptor
	settings     component.TelemetrySettings
	nextConsumer consumer.Metrics
}

func (m *mqttExporter) Shutdown(ctx context.Context) error {
	if m.adapter != nil {
		// this never returns an error
		_ = m.adapter.Disconnect()
		m.adapter = nil
	}

	return nil
}

func (m *mqttExporter) Start(_ context.Context, _ component.Host) error {
	m.adapter.SetAutoReconnect(true)
	m.adapter.SetCleanSession(false)

	if err := m.adapter.Connect(); err != nil {
		return err
	}

	return nil
}

func (m *mqttExporter) consumeMetrics(_ context.Context, metrics pmetric.Metrics) error {
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		resourceMetrics := metrics.ResourceMetrics().At(i)

		resourceAttributes := resourceMetrics.Resource().Attributes()

		for j := 0; j < resourceMetrics.ScopeMetrics().Len(); j++ {
			scopeMetrics := resourceMetrics.ScopeMetrics().At(j)
			for k := 0; k < scopeMetrics.Metrics().Len(); k++ {
				metric := scopeMetrics.Metrics().At(k)

				// for now, we know everything we care about is a gauge
				if metric.Type() != pmetric.MetricTypeGauge {
					continue
				}

				gauge := metric.Gauge()

				for l := 0; l < gauge.DataPoints().Len(); l++ {
					datapoint := gauge.DataPoints().At(l)

					mqttValue := make(map[string]any, resourceAttributes.Len()+datapoint.Attributes().Len())

					resourceAttributes.Range(func(k string, v pcommon.Value) bool {
						mqttValue[k] = v.AsString()
						return true
					})

					datapoint.Attributes().Range(func(k string, v pcommon.Value) bool {
						mqttValue[k] = v.AsString()
						return true
					})

					switch datapoint.ValueType() {
					case pmetric.NumberDataPointValueTypeDouble:
						value := datapoint.DoubleValue()
						if !math.IsNaN(value) {
							mqttValue["value"] = value
						}
					case pmetric.NumberDataPointValueTypeInt:
						mqttValue["value"] = datapoint.IntValue()
					case pmetric.NumberDataPointValueTypeEmpty:
						// mqttValue["value"] = math.NaN()
					}

					data, err := json.Marshal(mqttValue)
					if err != nil {
						m.settings.Logger.Warn(
							"failed to marshal value",
							zap.String("name", metric.Name()),
							zap.Error(err),
						)
						continue
					}

					topic := metric.Name()
					topic = strings.Replace(topic, ".", "/", -1)
					topic = strings.Replace(topic, " ", "_", -1)
					topic = strings.ToLower(topic)

					m.adapter.Publish(topic, data)
				}
			}
		}
	}
	return nil
}

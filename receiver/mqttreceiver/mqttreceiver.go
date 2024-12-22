package mqttreceiver

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	gojsonq "github.com/thedevsaddam/gojsonq/v2"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var typeStr = component.MustNewType("mqtt")

type MetricType string

const (
	GaugeMetricType   MetricType = "gauge"
	CounterMetricType MetricType = "counter"
)

func (t *MetricType) UnmarshalText(text []byte) error {
	v := strings.TrimSpace(strings.ToLower(string(text)))

	switch v {
	case "gauge", "":
		*t = GaugeMetricType
	case "counter":
		*t = CounterMetricType
	default:
		return fmt.Errorf("unsupported MetricType %q", v)
	}

	return nil
}

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Endpoint string `mapstructure:"endpoint"`
	ClientID string `mapstructure:"client_id"`
}

type TopicConfig struct {
	Path    string         `mapstructure:"path"`
	Metrics []MetricConfig `mapstructure:"metrics"`
}

type MetricConfig struct {
	AttributesRegex *regexp.Regexp    `mapstructure:"attributes_regex"`
	MetricName      string            `mapstructure:"metric_name"`
	PayloadPath     string            `mapstructure:"payload_path"`
	PathSeparator   string            `mapstructure:"path_separator"`
	Attributes      map[string]string `mapstructure:"attributes"`
	MetricType      MetricType        `mapstructure:"metric_type"`
}

type config struct {
	path    string
	metrics []metricConfig
}

type metricConfig struct {
	attributesRegex *regexp.Regexp
	metricName      string
	metricType      MetricType
	pathMatcher     *gojsonq.JSONQ
	attributes      map[string]*gojsonq.JSONQ
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
		ClientID: "otel-mqtt-receiver",
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
	adapter      *mqtt.Adaptor
	cancelFunc   context.CancelFunc
	settings     component.TelemetrySettings
	nextConsumer consumer.Metrics
	config       *config
}

func (m *mqttReceiver) Shutdown(ctx context.Context) error {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.adapter != nil {
		// this never returns an error
		_ = m.adapter.Disconnect()
		m.adapter = nil
	}

	return nil
}

func (m *mqttReceiver) Start(_ context.Context, _ component.Host) error {
	var client mqtt.Client

	_ = client

	return nil
}

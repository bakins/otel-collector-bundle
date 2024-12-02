package starlinkreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/bakins/otel-collector-bundle/receiver/starlinkreceiver/gen/spacex/api/device"
)

var typeStr = component.MustNewType("starlink")

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	configgrpc.ClientConfig        `mapstructure:",squash"`
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
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = 20 * time.Second

	return &Config{
		ControllerConfig: cfg,
		ClientConfig: configgrpc.ClientConfig{
			Endpoint: "192.168.100.1:9200",
		},
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	ns := newStarlinkScraper(params, cfg)
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

type starlinkScraper struct {
	settings   component.TelemetrySettings
	cfg        *Config
	clientConn *grpc.ClientConn
}

func newStarlinkScraper(settings receiver.Settings, cfg *Config) *starlinkScraper {
	s := &starlinkScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
	}

	return s
}

func (s *starlinkScraper) shutdown(_ context.Context) error {
	if s.clientConn == nil {
		return nil
	}

	err := s.clientConn.Close()

	s.clientConn = nil

	return err
}

func (s *starlinkScraper) start(ctx context.Context, host component.Host) error {
	clientConn, err := s.cfg.ClientConfig.ToClientConn(ctx, host, s.settings)
	if err != nil {
		return err
	}
	s.clientConn = clientConn
	return nil
}

func (s *starlinkScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	if s.clientConn == nil {
		return pmetric.Metrics{}, errors.New("grpc not connected")
	}

	client := device.NewDeviceClient(s.clientConn)
	resp, err := client.Handle(
		ctx,
		&device.Request{
			Request: &device.Request_GetHistory{
				GetHistory: &device.GetHistoryRequest{},
			},
		},
	)
	if err != nil {
		s.settings.Logger.Error("failed to fetch starlink stats", zap.Error(err))
		return pmetric.Metrics{}, err
	}

	history := resp.GetDishGetHistory()
	if history == nil {
		s.settings.Logger.Error("no starlink history present", zap.Error(err))
		return pmetric.Metrics{}, err
	}

	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	slice := scopeMetrics.Metrics()

	for name, samples := range map[string][]float32{
		"starlink.downlink.throughput": history.DownlinkThroughputBps,
		"starlink.uplink.throughput":   history.UplinkThroughputBps,
		"starlink.power_in":            history.PowerIn,
		"starlink.latency":             history.PopPingLatencyMs,
	} {
		// based on https://github.com/jim-olsen/SveltePowerMeter/blob/master/src/python/Starlink.py#L139
		// not sure if is correct, or if I copied logic right, but close enough for my needs

		numSamples := len(samples)
		loopEnd := int(history.Current) % numSamples
		value := samples[(loopEnd+1+0)%numSamples]

		m := slice.AppendEmpty()
		m.SetName(name)
		m.SetDescription(name)
		g := m.SetEmptyGauge()
		point := g.DataPoints().AppendEmpty()
		point.SetDoubleValue(float64(value))
	}

	return metrics, nil
}

package velairreceiver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/zap"

	"github.com/bakins/velair"
)

var typeStr = component.MustNewType("velair")

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	confighttp.ClientConfig        `mapstructure:",squash"`
}

func (cfg *Config) Validate() error {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: '%s': %w", cfg.Endpoint, err)
	}

	if u.Hostname() == "" {
		return fmt.Errorf("missing hostname: '%s'", cfg.Endpoint)
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

	clientConfig := confighttp.NewDefaultClientConfig()
	clientConfig.Timeout = 10 * time.Second

	return &Config{
		ControllerConfig: cfg,
		ClientConfig:     clientConfig,
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	ns := &velairScraper{
		settings: params.TelemetrySettings,
		cfg:      cfg,
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		consumer,
		scraperhelper.AddScraper(typeStr, ns),
	)
}

type velairScraper struct {
	settings component.TelemetrySettings
	cfg      *Config
	client   *velair.Client
}

func (s *velairScraper) Shutdown(_ context.Context) error {
	return nil
}

func (s *velairScraper) Start(ctx context.Context, host component.Host) error {
	httpClient, err := s.cfg.ToClient(ctx, host, s.settings)
	if err != nil {
		return err
	}

	c, err := velair.New(s.cfg.Endpoint, velair.WithDoer(httpClient))
	if err != nil {
		return err
	}

	s.client = c

	return nil
}

func (s *velairScraper) ScrapeMetrics(ctx context.Context) (pmetric.Metrics, error) {
	if s.client == nil {
		return pmetric.Metrics{}, errors.New("http client not configured")
	}

	metrics := pmetric.NewMetrics()

	status, err := s.client.GetStatus(ctx)
	if err != nil {
		s.settings.Logger.Error("failed to get velair status", zap.Error(err))
		return metrics, nil
	}

	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()

	if name := status.Name; name != "" {
		resourceAttributes := resourceMetrics.Resource().Attributes()
		resourceAttributes.PutStr("local_name", name)
	}

	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	slice := scopeMetrics.Metrics()

	m := slice.AppendEmpty()
	m.SetName("velair.temperature")
	m.SetDescription("ambient temperature in C")
	g := m.SetEmptyGauge()
	point := g.DataPoints().AppendEmpty()
	point.SetIntValue(int64(status.Temperature))
	return metrics, nil
}

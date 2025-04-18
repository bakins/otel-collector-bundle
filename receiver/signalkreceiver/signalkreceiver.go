package signalkreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Endpoint string   `mapstructure:"endpoint"`
	Paths    []string `mapstructure:"paths"`
}

// Validate checks if the receiver configuration is valid
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

var typeStr = component.MustNewType("signalk")

// NewFactory creates a factory for ruuvitag receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha))
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createMetricsReceiver(_ context.Context, params receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	config := baseCfg.(*Config)

	r := &signalkReceiver{
		settings:     params.TelemetrySettings,
		nextConsumer: consumer,
		config:       config,
	}

	return r, nil
}

type signalkReceiver struct {
	settings     component.TelemetrySettings
	cancel       context.CancelFunc
	config       *Config
	nextConsumer consumer.Metrics
}

func (r *signalkReceiver) Start(_ context.Context, host component.Host) error {
	ctx := context.Background()
	ctx, r.cancel = context.WithCancel(context.Background())

	go func() {
		r.receiveMetrics(ctx)
	}()

	return nil
}

func (r *signalkReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *signalkReceiver) receiveMetrics(ctx context.Context) {
	err := retry.Do(func() error {
		err := r.doReceiveMetrics(ctx)
		switch {
		case errors.Is(err, context.Canceled):
			return nil
		// this generally means the context inside the function was canceled
		// the check for the parent context above will catch the outer cancel
		default:
			return err
		}
	},
		retry.Context(ctx),
		retry.MaxDelay(time.Second*5),
		retry.MaxJitter(time.Second),
		retry.Attempts(0),
		retry.LastErrorOnly(true),
		retry.OnRetry(
			func(attempt uint, err error) {
				r.settings.Logger.Warn(
					"signalk retry",
					zap.Uint("attempt", attempt),
					zap.Error(err),
				)
			},
		),
	)
	if err != nil {
		r.settings.Logger.Error("signalk fatal", zap.Error(err))
	}
}

func (r *signalkReceiver) doReceiveMetrics(ctx context.Context) error {
	conn, resp, err := websocket.Dial(ctx, "ws://"+r.config.Endpoint+"/signalk/v1/stream?subscribe=none", nil)
	if err != nil {
		return err
	}

	defer conn.CloseNow()
	if resp.StatusCode != 101 {
		return fmt.Errorf("unexpected http code %d", resp.StatusCode)
	}

	// need to read server connection data first
	_, _, err = conn.Read(ctx)
	if err != nil {
		return err
	}

	req := subscriptionRequest{
		Context: "vessels.self",
	}

	for _, p := range r.config.Paths {
		req.Subscribe = append(req.Subscribe,
			subscription{
				Path:   p,
				Period: 5 * 1000,
			},
		)
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()
		_ = conn.Close(websocket.StatusNormalClosure, "")

		return nil
	})

	eg.Go(func() error {
		// subscriptions will start flowing as soon as this returns
		return wsjson.Write(ctx, conn, &req)
	})

	eg.Go(func() error {
		return r.websocketWorker(ctx, conn)
	})

	return eg.Wait()
}

func (r *signalkReceiver) websocketWorker(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var message deltaMessage

			if err := wsjson.Read(ctx, conn, &message); err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return nil
			default:
				r.handleDeltaMessage(context.Background(), message)
			}
		}
	}
}

func (r *signalkReceiver) handleDeltaMessage(ctx context.Context, message deltaMessage) {
	for _, update := range message.Updates {
		metrics := pmetric.NewMetrics()
		resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
		attributes := resourceMetrics.Resource().Attributes()
		attributes.PutStr("source_src", update.Source.Src)
		attributes.PutStr("source_label", update.Source.Label)
		attributes.PutStr("source_pgn", strconv.Itoa(update.Source.Pgn))

		slice := resourceMetrics.ScopeMetrics().AppendEmpty().Metrics()

		for _, value := range update.Values {
			if value.Value == nil {
				continue
			}

			v, ok := value.Value.(float64)
			if !ok {
				continue
			}

			m := slice.AppendEmpty()
			m.SetName("signalk." + strings.ToLower(value.Path))
			m.SetDescription("signalk " + value.Path)
			point := m.SetEmptyGauge().DataPoints().AppendEmpty()
			point.SetDoubleValue(v)
		}

		if slice.Len() == 0 {
			continue
		}

		if err := r.nextConsumer.ConsumeMetrics(ctx, metrics); err != nil {
			r.settings.Logger.Warn("failed to pass along metrics", zap.Error(err))
		}
	}
}

type subscriptionRequest struct {
	Context   string         `json:"context"`
	Subscribe []subscription `json:"subscribe"`
}

type subscription struct {
	Path   string `json:"path"`
	Period int    `json:"period"`
}

type deltaMessage struct {
	Context string  `json:"context"`
	Updates []delta `json:"updates"`
}

type delta struct {
	Source deltaSource  `json:"source"`
	Values []deltaValue `json:"values"`
}

type deltaSource struct {
	Label          string `json:"label"`
	Src            string `json:"src"`
	Pgn            int    `json:"pgn"`
	DeviceInstance int    `json:"deviceInstance"` // always zero?
}

type deltaValue struct {
	Path  string `json:"path"`
	Value any    `json:"value"`
}

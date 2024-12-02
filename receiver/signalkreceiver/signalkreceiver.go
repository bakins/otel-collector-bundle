package signalkreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
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
	ctx, r.cancel = context.WithCancel(ctx)

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
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := r.doReceiveMetrics(ctx)
			switch {
			case errors.Is(err, context.Canceled):
				// this generally means the context inside the function was canceled
				// the check for the parent context above will catch the outer cancel
			case strings.Contains(err.Error(), "use of closed network connection"):
				// usually means websocket was closed, fine to continue
			default:
				r.settings.Logger.Warn("failed to collect signalk metrics", zap.Error(err))
				time.Sleep(time.Second)
			}
		}
	}
}

func (r *signalkReceiver) doReceiveMetrics(ctx context.Context) error {
	// occasionally the stream of events just drops, so reconnect periodically.
	// the stream tends to just hang :shrug:
	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	d := &websocket.Dialer{}
	conn, resp, err := d.DialContext(ctx, "ws://"+r.config.Endpoint+"/signalk/v1/stream?subscribe=none", nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != 101 {
		return fmt.Errorf("unexpected http code %d", resp.StatusCode)
	}

	// need to read server connection data first
	_, _, err = conn.ReadMessage()
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
		_ = conn.Close()

		return nil
	})

	eg.Go(func() error {
		// subscriptions will start flowing as soon as this returns
		if err := conn.WriteJSON(&req); err != nil {
			return err
		}

		return nil
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
			_, reader, err := conn.NextReader()
			if err != nil {
				return err
			}

			var message deltaMessage
			if err := json.NewDecoder(reader).Decode(&message); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				slog.ErrorContext(ctx, "failed to read json from websocket", "error", err)
				continue
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

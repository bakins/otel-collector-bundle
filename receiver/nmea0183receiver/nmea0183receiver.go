package nmea0183receiver

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	nmea "github.com/adrianmo/go-nmea"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"golang.org/x/sync/errgroup"
)

var typeStr = component.MustNewType("nmea0183")

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Port uint16 `mapstructure:"port"`
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
	return &Config{
		Port: 4000,
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", cfg.Port))
	if err != nil {
		return nil, err
	}

	return &nmea0183Receiver{
		addr: addr,
		next: consumer,
	}, nil
}

type nmea0183Receiver struct {
	addr         *net.UDPAddr
	shutdownFunc func()
	next         consumer.Metrics
}

func (r *nmea0183Receiver) Start(_ context.Context, host component.Host) error {
	conn, err := net.ListenUDP("udp", r.addr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	eg, ctx := errgroup.WithContext(ctx)

	r.shutdownFunc = cancel

	sentences := make(chan nmea.Sentence, 8)

	eg.Go(func() error {
		<-ctx.Done()
		conn.Close()
		return nil
	})

	eg.Go(func() error {
		buffer := make([]byte, 2048)

		for {
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {

				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				// check if temporary?
				return err
			}

			text := strings.ReplaceAll(string(buffer[:n]), "\r\n", "\n")
			lines := strings.Split(text, "\n")

			for _, line := range lines {
				if len(line) == 0 {
					continue
				}

				sentence, err := nmea.Parse(line)
				if err != nil {
					// log?
					continue
				}

				sentences <- sentence
			}
		}
	})

	eg.Go(func() error {
		return handleSentences(ctx, r.next, sentences)
	})

	go func() {
		err := eg.Wait()
		if err != nil && err != context.Canceled {
			// log error?
		}
		close(sentences)
	}()

	return nil
}

var sentenceHandlers = map[string]func(context.Context, nmea.Sentence) (pmetric.Metrics, error){
	nmea.TypeMTW: handleWaterTemperature,
	nmea.TypeDPT: handleDepthBelowKeel,
	nmea.TypeVWR: handleApparentWind,
	nmea.TypeVWT: handleTrueWind,
}

func addCommonAttributes(attributes pcommon.Map, sentence nmea.Sentence) {
	attributes.PutStr("talker", sentence.TalkerID())

	// this is terrible...
	v := reflect.ValueOf(sentence)
	if v.Kind() != reflect.Struct {
		return
	}

	field := v.FieldByName("BaseSentence")
	if field.IsZero() {
		return
	}

	base, ok := field.Interface().(nmea.BaseSentence)
	if !ok {
		return
	}

	attributes.PutStr("source", base.TagBlock.Source)
	attributes.PutStr("text", base.TagBlock.Text)
}

func handleWaterTemperature(_ context.Context, sentence nmea.Sentence) (pmetric.Metrics, error) {
	m, ok := sentence.(nmea.MTW)
	if !ok {
		return pmetric.Metrics{}, nil
	}

	if !m.CelsiusValid {
		return pmetric.Metrics{}, nil
	}

	metrics := pmetric.NewMetrics()
	metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("nmea0183_water_temperature")
	metric.SetDescription("water temperature surronding vessel")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(m.Temperature)

	return metrics, nil
}

func handleDepthBelowKeel(_ context.Context, sentence nmea.Sentence) (pmetric.Metrics, error) {
	m, ok := sentence.(nmea.DPT)
	if !ok {
		return pmetric.Metrics{}, nil
	}

	metrics := pmetric.NewMetrics()
	metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("nmea0183_depth_below_keel")
	metric.SetDescription("water depth below keel")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(m.Depth + m.Offset)

	return metrics, nil
}

func handleApparentWind(_ context.Context, sentence nmea.Sentence) (pmetric.Metrics, error) {
	m, ok := sentence.(nmea.VWR)
	if !ok {
		return pmetric.Metrics{}, nil
	}

	metrics := pmetric.NewMetrics()
	scopeMetrics := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
	speed := scopeMetrics.AppendEmpty()
	angle := scopeMetrics.AppendEmpty()
	speed.SetName("nmea0183_wind_speed_apparent")
	angle.SetName("nmea0183_wind_angle_apparent")

	angle.SetDescription("apparent wind angle in relation to boat")

	val := m.MeasuredAngle
	if m.MeasuredDirectionBow == "L" {
		val = -val
	}

	speed.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(m.SpeedMPS)
	angle.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(val)

	return metrics, nil
}

func handleTrueWind(_ context.Context, sentence nmea.Sentence) (pmetric.Metrics, error) {
	m, ok := sentence.(nmea.VWT)
	if !ok {
		return pmetric.Metrics{}, nil
	}

	metrics := pmetric.NewMetrics()
	scopeMetrics := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics()
	speed := scopeMetrics.AppendEmpty()
	angle := scopeMetrics.AppendEmpty()
	speed.SetName("nmea0183_wind_speed_true")
	angle.SetName("nmea0183_wind_angle_true")
	angle.SetDescription("true wind angle in relation to boat")

	speed.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(m.SpeedMPS)

	val := m.TrueAngle
	if m.TrueDirectionBow == "L" {
		val = -val
	}

	angle.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(val)

	return metrics, nil
}

func handleSentences(ctx context.Context, next consumer.Metrics, sentences <-chan nmea.Sentence) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				err = nil
			}

			return err
		case sentence, ok := <-sentences:
			if !ok {
				return nil
			}

			_ = sentence

			handler, ok := sentenceHandlers[sentence.DataType()]
			if !ok {
				continue
			}

			metrics, err := handler(ctx, sentence)
			if err != nil {
				// log
				continue
			}

			if metrics.MetricCount() == 0 {
				continue
			}

			resourceMetrics := metrics.ResourceMetrics()
			for i := 0; i < resourceMetrics.Len(); i++ {
				r := resourceMetrics.At(i).Resource()
				addCommonAttributes(r.Attributes(), sentence)
			}

			if err := next.ConsumeMetrics(ctx, metrics); err != nil {
				// log???
			}
		}
	}
}

func (r *nmea0183Receiver) Shutdown(_ context.Context) error {
	if r.shutdownFunc != nil {
		r.shutdownFunc()
	}

	return nil
}

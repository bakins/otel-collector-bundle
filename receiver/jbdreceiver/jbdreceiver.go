package jbdreceiver

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	retry "github.com/avast/retry-go/v4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"tinygo.org/x/bluetooth"

	"github.com/bakins/otel-collector-bundle/bleutil"
)

var typeStr = component.MustNewType("jbd")

type Config struct {
	Address string `mapstructure:"address"`
}

type bluetoothAddress struct {
	bluetooth.Address
}

func (cfg *Config) Validate() error {
	if cfg.Address == "" {
		return errors.New("address is required")
	}

	var b bluetoothAddress

	if err := b.UnmarshalText([]byte(cfg.Address)); err != nil {
		return fmt.Errorf("invalid bluetooth address: %w", err)
	}

	return nil
}

type jbdReceiver struct {
	cancel       context.CancelFunc
	config       *Config
	address      bluetooth.Address
	nextConsumer consumer.Metrics
	scanner      bleutil.Scanner
	logger       *zap.Logger
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha))
}

func createMetricsReceiver(_ context.Context, params receiver.Settings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	config := baseCfg.(*Config)

	var b bluetoothAddress
	if err := b.UnmarshalText([]byte(config.Address)); err != nil {
		return nil, err
	}

	r := &jbdReceiver{
		logger:       params.Logger.With(zap.String("address", b.Address.String())),
		nextConsumer: consumer,
		config:       config,
		address:      b.Address,
	}

	return r, nil
}

func (r *jbdReceiver) Start(_ context.Context, host component.Host) error {
	ctx := context.Background()
	ctx, r.cancel = context.WithCancel(ctx)

	if r.scanner == nil {
		scanner, err := bleutil.DefaultScanner()
		if err != nil {
			return err
		}
		r.scanner = scanner
	}

	go r.worker(ctx)

	return nil
}

func (r *jbdReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *jbdReceiver) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := r.handleBattery(ctx); err != nil {
				r.logger.Warn("failed to handle battery metrics",
					zap.Error(err),
					zap.Stringer("address", r.address),
				)
			}
			time.Sleep(time.Second)
		}
	}
}

func mustParse(s string) bluetooth.UUID {
	u, err := bluetooth.ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return u
}

var (
	bmsServiceUUID               = mustParse("0000ff00-0000-1000-8000-00805f9b34fb")
	bmsSendCharacteristicUUID    = mustParse("0000ff02-0000-1000-8000-00805f9b34fb")
	bmsReceiveCharacteristicUUID = mustParse("0000ff01-0000-1000-8000-00805f9b34fb")
)

// see https://github.com/neilsheps/overkill-xiaoxiang-jbd-bms-ble-reader/blob/main/src/main.cpp
const (
	responseGeneralInfo  = 0x03
	responseCellVoltages = 0x04
)

var (
	requestGeneralInfo  = []byte{0xdd, 0xa5, responseGeneralInfo, 0x00, 0xff, 0xfd, 0x77}
	requestCellVoltages = []byte{0xdd, 0xa5, responseCellVoltages, 0x0, 0xff, 0xfc, 0x77}
)

func (r *jbdReceiver) discoverBattery(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var found bool
	r.scanner.Scan(ctx, func(rc bleutil.ScanResult) {
		if rc.Address == r.address.String() {
			found = true
			cancel()
		}
	})

	if !found {
		return fmt.Errorf("failed to discover battery %s", r.address.String())
	}

	return nil
}

func (r *jbdReceiver) handleBattery(ctx context.Context) error {
	if err := r.discoverBattery(ctx); err != nil {
		return err
	}

	adapter := bluetooth.DefaultAdapter

	var conn bluetooth.Device
	err := retry.Do(func() error {
		c, err := adapter.Connect(r.address, bluetooth.ConnectionParams{
			ConnectionTimeout: bluetooth.NewDuration((time.Second * 10)),
			Timeout:           bluetooth.NewDuration((time.Second * 10)),
		})
		if err != nil {
			return fmt.Errorf("failed to connect %w", err)
		}

		conn = c

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(10),
		retry.Delay(time.Second),
		retry.LastErrorOnly(true),
		retry.MaxDelay(time.Second*3),
	)
	if err != nil {
		return err
	}

	defer conn.Disconnect()

	services, err := conn.DiscoverServices([]bluetooth.UUID{bmsServiceUUID})
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		return fmt.Errorf("no services discovered")
	}

	characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{bmsSendCharacteristicUUID, bmsReceiveCharacteristicUUID})
	if err != nil {
		return fmt.Errorf("failed to discover characteristics: %w", err)
	}

	if len(characteristics) != 2 {
		return fmt.Errorf("failed to discover characteristics")
	}

	var (
		writer bluetooth.DeviceCharacteristic
		reader bluetooth.DeviceCharacteristic
	)

	for _, char := range characteristics {
		switch char.UUID() {
		case bmsSendCharacteristicUUID:
			writer = char
		case bmsReceiveCharacteristicUUID:
			reader = char
		}
	}

	if writer == (bluetooth.DeviceCharacteristic{}) {
		return fmt.Errorf("failed to find writer")
	}

	if reader == (bluetooth.DeviceCharacteristic{}) {
		return fmt.Errorf("failed to find reader")
	}

	generalCallback := func(data []byte) {
		metrics := pmetric.NewMetrics()

		slice := metrics.ResourceMetrics().AppendEmpty().
			ScopeMetrics().AppendEmpty().
			Metrics()

		v := float64(binary.BigEndian.Uint16(data[:2])) / 100.0

		voltage := slice.AppendEmpty()
		voltage.SetName("jbd.battery.voltage")
		voltage.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(v)

		c := float64(int16(binary.BigEndian.Uint16(data[2:4]))) / 100.0

		current := slice.AppendEmpty()
		current.SetName("jbd.battery.current")
		current.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(c)

		s := int64(data[19])

		soc := slice.AppendEmpty()
		soc.SetName("jbd.battery.soc")
		soc.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(s)

		temperatureCount := int(data[22])
		var temperatureSum float64
		for i := 0; i < temperatureCount; i++ {
			index := 23 + (i * 2)
			temperature := float64(binary.BigEndian.Uint16(data[index:index+2])-2731) / 10.0
			temperatureSum += temperature
		}

		temperature := slice.AppendEmpty()
		temperature.SetName("jbd.battery.temperature")
		temperature.SetEmptyGauge().DataPoints().AppendEmpty().SetDoubleValue(temperatureSum / float64(temperatureCount))

		if err := r.nextConsumer.ConsumeMetrics(context.Background(), metrics); err != nil {
			r.logger.Warn("failed to pass along metrics", zap.Error(err))
		}
	}

	cellVoltageCallback := func(data []byte) {
		numCells := len(data) / 2

		metrics := pmetric.NewMetrics()

		slice := metrics.ResourceMetrics().AppendEmpty().
			ScopeMetrics().AppendEmpty().
			Metrics()

		voltage := slice.AppendEmpty()
		voltage.SetName("jbd.battery.cell.voltage")

		for i := 0; i < numCells; i++ {
			millivolts := float64(binary.BigEndian.Uint16(data[2*i : 2*i+2]))
			// millivolts := int(data[2*i])*256 + int(data[1+2*i])
			v := float64(millivolts / 1000.0)

			point := voltage.SetEmptyGauge().DataPoints().AppendEmpty()
			point.Attributes().PutStr("cell", strconv.Itoa(i))
			point.SetDoubleValue(v)
		}

		if err := r.nextConsumer.ConsumeMetrics(context.Background(), metrics); err != nil {
			r.logger.Warn("failed to pass along metrics", zap.Error(err))
		}
	}

	responseHandler := &responseBuffer{
		callback: func(dataType int, data []byte) {
			if dataType == responseGeneralInfo {
				generalCallback(data)
			} else {
				cellVoltageCallback(data)
			}
		},
		errorCallback: func(err error) {
			r.logger.Warn("device communication error", zap.Error(err))
		},
	}

	if err := reader.EnableNotifications(responseHandler.handleNotification); err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	defer func() {
		_ = reader.EnableNotifications(nil)
	}()

	sendRequest := func(request []byte) error {
		if _, err := writer.WriteWithoutResponse(request); err != nil {
			return fmt.Errorf("failed to write value: %w", err)
		}

		return nil
	}

	ticker := time.NewTimer(time.Second * 20)
	defer ticker.Stop()

	requests := [][]byte{
		requestGeneralInfo,
		requestCellVoltages,
	}

	var current int

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sendRequest(requests[current]); err != nil {
				return fmt.Errorf("failed to send request: %w", err)
			}

			current = (current + 1) % 2
		}
	}
}

const (
	stateFirstRead = 0
	stateReadData  = 1
)

type responseBuffer struct {
	buffer        bytes.Buffer
	callback      func(dataType int, data []byte)
	errorCallback func(err error)
	state         int
	dataType      int
	expected      int
	received      int
}

func (r *responseBuffer) reset() {
	r.buffer.Reset()
	r.state = stateFirstRead
	r.dataType = 0
	r.expected = 0
	r.received = 0
}

func (r *responseBuffer) handleNotification(data []byte) {
	if len(data) == 0 {
		r.errorCallback(fmt.Errorf("empty notification"))
		r.reset()
		return
	}

	if len(data) == 1 {
		r.errorCallback(fmt.Errorf("empty notification"))
		r.reset()
		return
	}

	switch r.state {
	case stateFirstRead:
		switch data[1] {
		case responseGeneralInfo:
		case responseCellVoltages:
		default:
			r.errorCallback(fmt.Errorf("unexpected response type: %X", data[1]))
			r.reset()
			return
		}

		if data[0] != 0xdd {
			r.reset()
			r.errorCallback(fmt.Errorf("invalid start byte: %X", data[0]))
			return
		}

		if data[2] != 0 {
			r.errorCallback(fmt.Errorf("an error occurred: %X", data[2]))
			r.reset()
			return
		}

		r.dataType = int(data[1])

		// length of data, plus header/footer
		r.expected = int(data[3]) + 7

		// did we get it all?
		if len(data) >= r.expected {
			if !checkChecksum(data) {
				r.errorCallback(fmt.Errorf("checksum failed"))
				r.reset()
				return
			}

			r.callback(r.dataType, data[4:len(data)-3])
			r.reset()

			return
		}

		r.received = len(data)

		if _, err := r.buffer.Write(data); err != nil {
			r.errorCallback(err)
			r.reset()
			return
		}

		r.state = stateReadData

	case stateReadData:
		left := r.expected - r.received

		if len(data) >= left {
			if _, err := r.buffer.Write(data[:left]); err != nil {
				r.errorCallback(err)
				r.reset()
				return
			}

			if !checkChecksum(r.buffer.Bytes()) {
				r.errorCallback(fmt.Errorf("checksum failed"))
				r.reset()
				return
			}

			r.callback(r.dataType, r.buffer.Bytes()[4:r.buffer.Len()-3])
			r.reset()
			return
		}

		r.received += len(data)
		if _, err := r.buffer.Write(data); err != nil {
			r.errorCallback(err)
			r.reset()
			return
		}
	}
}

func checkChecksum(data []byte) bool {
	checksumIndex := (int)((data[3]) + 4)

	checksum := getChecksumForReceivedData(data)
	expected := int(data[checksumIndex])*256 + int(data[checksumIndex+1])

	return checksum == uint16(expected)
}

func getChecksumForReceivedData(data []byte) uint16 {
	checksum := 0x10000
	dataLengthProvided := int(data[3])

	for i := 0; i < dataLengthProvided+1; i++ {
		checksum -= int(data[i+3])
	}
	return uint16(checksum)
}

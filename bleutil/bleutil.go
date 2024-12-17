package bleutil

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bakins/bluetooth"
)

func Scan(
	ctx context.Context,
	adapter *bluetooth.Adapter,
	callback func(ctx context.Context, result bluetooth.ScanResult) bool,
) error {
	running := &atomic.Bool{}
	running.Store(true)

	// so the go routing waiting on context can finish
	done := make(chan struct{})

	stopScan := func() {
		if running.CompareAndSwap(true, false) {
			// only returns an error if not scanning
			// must only call once
			_ = adapter.StopScan()
			close(done)
		}
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		select {
		case <-ctx.Done():
			stopScan()
		case <-done:
			stopScan()
		}

		return nil
	})

	gotMessage := make(chan time.Time)

	eg.Go(func() error {
		// if we don't get anything within 60 seconds, assume bluetooth failed
		// it happens a lot on my raspberry pi
		ticker := time.NewTicker(time.Second * 20)
		defer ticker.Stop()

		lastUpdate := time.Now()

		for {
			select {
			case <-ctx.Done():
				return nil
			case update := <-gotMessage:
				lastUpdate = update
			case now := <-ticker.C:
				if !running.Load() {
					return nil
				}

				if now.Sub(lastUpdate).Seconds() > 60.0 {
					stopScan()

					return fmt.Errorf("bluetooth scan failed")
				}
			}
		}
	})

	eg.Go(func() error {
		return adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			gotMessage <- time.Now()
			if !callback(ctx, result) {
				stopScan()
			}
		})
	})

	return eg.Wait()
}

type Scanner interface {
	Scan(ctx context.Context, callback func(ScanResult))
}

type ScanResult struct {
	Address          string
	LocalName        string
	ManufacturerData []ManufacturerDataElement
}

type ManufacturerDataElement struct {
	CompanyID uint16
	Data      []byte
}

type adapterScanner struct {
	mu      sync.Mutex
	adapter *bluetooth.Adapter
	counter uint64
	callers map[uint64]func(ScanResult)
	started bool
}

var defaultAdapterScanner = &adapterScanner{
	callers: map[uint64]func(ScanResult){},
}

func DefaultScanner() (Scanner, error) {
	defaultAdapterScanner.mu.Lock()
	defer defaultAdapterScanner.mu.Unlock()

	if defaultAdapterScanner.adapter != nil {
		return defaultAdapterScanner, nil
	}

	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, fmt.Errorf("unable to enable default bluetooth adapter %w", err)
	}

	defaultAdapterScanner.adapter = adapter

	return defaultAdapterScanner, nil
}

func (a *adapterScanner) startScan() {
	a.adapter.Scan(func(_ *bluetooth.Adapter, sr bluetooth.ScanResult) {
		result := ScanResult{
			Address:   sr.Address.String(),
			LocalName: sr.LocalName(),
		}

		for _, md := range sr.ManufacturerData() {
			result.ManufacturerData = append(result.ManufacturerData,
				ManufacturerDataElement{
					CompanyID: md.CompanyID,
					Data:      md.Data,
				},
			)
		}

		var callbacks []func(ScanResult)

		a.mu.Lock()
		for _, cb := range a.callers {
			callbacks = append(callbacks, cb)
		}
		a.mu.Unlock()

		for _, cb := range callbacks {
			cb(result)
		}
	})
}

func (a *adapterScanner) Scan(ctx context.Context, callback func(ScanResult)) {
	var id uint64

	a.mu.Lock()

	a.counter++
	id = a.counter
	a.callers[id] = callback

	if !a.started {
		go a.startScan()
		a.started = true
	}
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.callers, id)
		a.mu.Unlock()
	}()

	<-ctx.Done()
}

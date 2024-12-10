package bleutil

import (
	"context"
	"fmt"
	"sync"

	"tinygo.org/x/bluetooth"
)

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

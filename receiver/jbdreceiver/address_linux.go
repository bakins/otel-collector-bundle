package jbdreceiver

import "github.com/bakins/bluetooth"

func (b *bluetoothAddress) UnmarshalText(text []byte) error {
	mac, err := bluetooth.ParseMAC(string(text))
	if err != nil {
		return err
	}

	b.Address = bluetooth.Address{
		MACAddress: bluetooth.MACAddress{
			MAC: mac,
		},
	}

	return nil
}

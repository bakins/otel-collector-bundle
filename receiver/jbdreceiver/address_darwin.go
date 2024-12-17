package jbdreceiver

import "github.com/bakins/bluetooth"

func (b *bluetoothAddress) UnmarshalText(text []byte) error {
	uuid, err := bluetooth.ParseUUID(string(text))
	if err != nil {
		return err
	}

	b.Address = bluetooth.Address{
		UUID: uuid,
	}

	return nil
}

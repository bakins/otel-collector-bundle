package jbdreceiver

import "tinygo.org/x/bluetooth"

func (b *bluetoothAdress) UnmarshalText(text []byte) error {
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

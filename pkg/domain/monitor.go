package domain

import "math/big"

type (
	Monitor struct {
		Enabled bool
	}

	AddressMonitor struct {
		Monitor
		Addresses []NamedAddress
	}

	NamedAddress struct {
		Name string
		Addr string
	}

	BigTransfersMonitor struct {
		Monitor
		MinThreshold big.Int
	}
)

package main

import (
	"encoding/json"
	"os"
)

const (
	AddressBNB = "bnb"
	AddressBUSD = "busd"
	AddressCakeRouter = "cake_router"
	AddressCakeFactory = "cake_factory"
)

type (
	Config struct {
		Account Account `json:"account"`
		Addresses map[string]string `json:"addresses"`
		Sniper  Sniper `json:"sniper"`
		Monitors Monitors `json:"monitors"`
	}

	Account struct {
		Address string `json:"address"`
		PK string `json:"pk"`
	}

	Sniper struct {
		Trigger string `json:"trigger"`
		GasPrice int64 `json:"standard_gas_price"`
		BaseCurrency string `json:"base_currency"`
		TargetToken string `json:"target_token"`
		MinLiquidity int `json:"minimum_liquidity"`
	}

	Monitors struct {
		AddressListMonitor AddressListMonitor `json:"address_list"`
		BigTransfersMonitor BigTransfersMonitor `json:"big_transfers"`
	}

	AddressListMonitor struct {
		Enabled bool `json:"enabled"`
		List []AddressListEntry `json:"list"`
	}

	AddressListEntry struct {
		Name string `json:"name"`
		Addr string `json:"addr"`
	}

	BigTransfersMonitor struct {
		Enabled bool `json:"enabled"`
		Min     int `json:"min"`
	}
)

func NewConfigFromFile(f string) (*Config, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err = json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

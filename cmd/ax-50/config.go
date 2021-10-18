package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

const (
	AddressCakeRouter  AddressName = "cake_router"
	AddressCakeFactory AddressName = "cake_factory"
)

type (
	AddressName string

	Config struct {
		Addresses map[string]string `json:"addresses"`
		Sniper    Sniper            `json:"sniper"`
		Monitors  Monitors          `json:"monitors"`
	}

	Sniper struct {
		Trigger      string  `json:"trigger"`
		BaseCurrency string  `json:"base_currency"`
		TargetToken  string  `json:"target_token"`
		MinLiquidity float32 `json:"minimum_liquidity"`
		RPCRead      string  `json:"rpc_read"`
		RPCWrite     string  `json:"rpc_write"`
	}

	Monitors struct {
		AddressListMonitor AddressListMonitor `json:"address_list"`
		WhaleMonitor       WhaleMonitor       `json:"whale"`
	}

	AddressListMonitor struct {
		Enabled bool               `json:"enabled"`
		List    []AddressListEntry `json:"list"`
	}

	AddressListEntry struct {
		Name string `json:"name"`
		Addr string `json:"addr"`
	}

	WhaleMonitor struct {
		Enabled bool   `json:"enabled"`
		Min     string `json:"min"`
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

func (c *Config) Addr(name AddressName) (common.Address, error) {
	if v, ok := c.Addresses[string(name)]; ok {
		return common.HexToAddress(v), nil
	}
	return common.Address{}, fmt.Errorf("%s not found in config addressses", name)
}

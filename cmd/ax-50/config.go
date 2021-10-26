package main

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type (
	Address string

	Config struct {
		Chains    ChainContainer `json:"chain"`
		Contracts Contracts      `json:"contract"`
		Tokens    Tokens         `json:"token"`
		Sniper    Sniper         `json:"sniper"`
	}

	ChainContainer struct {
		Nodes ChainNodes `json:"nodes"`
		ID    uint       `json:"id"`
		Name  string     `json:"name"`
	}

	ChainNodes struct {
		Stream string   `json:"stream"`
		Snipe  []string `json:"snipe"`
	}

	Contracts struct {
		Trigger Address `json:"trigger"`
		Factory Address `json:"factory"`
		Router  Address `json:"router"`
	}

	Tokens struct {
		SnipeA Address `json:"address"`
		SnipeB Address `json:"pair_address"`
		WBNB   Address `json:"wbnb"`
	}

	Sniper struct {
		MinLiquidity float32  `json:"minimum_liquidity"`
		Monitors     Monitors `json:"monitors"`
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
		Name string  `json:"name"`
		Addr Address `json:"addr"`
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

func (a Address) Addr() common.Address {
	if len(a) == 0 {
		panic("empty address")
	}
	return common.HexToAddress(string(a))
}

func (a Address) Hex() string {
	if len(a) == 0 {
		panic("empty address")
	}
	return string(a)
}

package service

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type (
	WhaleMonitor struct {
		MinThreshold *big.Int
	}
)

func NewWhaleMonitor(m *big.Int) *WhaleMonitor {
	return &WhaleMonitor{
		MinThreshold: m,
	}
}

func (m *WhaleMonitor) Monitor(ctx context.Context, tx *types.Transaction) {
	if tx.Value().Cmp(m.MinThreshold) == 1 {
		log.Info(fmt.Sprintf(
			"[WhaleMonitor] Transfer detected:\n    Hash: %v\n    Value: %v",
			tx.Hash().Hex(),
			formatETHWeiToEther(tx.Value()),
		))
	}
}

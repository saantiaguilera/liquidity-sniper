package usecase

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/saantiaguilera/liquidity-ax-50/pkg/domain"
	"sync"
)

var (
	// TODO Use for strategy mapping!
	addLiquidityETH = [...]byte{0xf3, 0x05, 0xd7, 0x19}
	addLiquidity = [...]byte{0xe8, 0xe3, 0x37, 0x00}
)

type (
	TransactionClassifier struct {
		mut *sync.Mutex // TODO: Check if necessary

		routerAddr string

		monitor transactionClassifierMonitor
		strategies map[[4]byte]transactionClassifierStrategy
	}

	transactionClassifierMonitor func(ctx context.Context, tx *types.Transaction)
	transactionClassifierStrategy func(ctx context.Context, tx *types.Transaction) error
)

func NewTransactionClassifier(
	raddr string,
	m transactionClassifierMonitor,
	s map[[4]byte]transactionClassifierStrategy,
) *TransactionClassifier {

	return &TransactionClassifier{
		mut:        new(sync.Mutex),
		routerAddr: raddr,
		monitor:    m,
		strategies: s,
	}
}

func (u *TransactionClassifier) Classify(ctx context.Context, tx *types.Transaction) error {
	if tx.To() == nil {
		return domain.ErrTxIsContract
	}

	u.monitor(ctx, tx)

	if tx.To().Hex() == u.routerAddr && len(tx.Data()) >= 4 {
		u.mut.Lock() // TODO Check need. I think the sniper service already handles these lock.
		defer u.mut.Unlock()

		txFunctionHash := [4]byte{}
		copy(txFunctionHash[:], tx.Data()[:4])

		if h, ok := u.strategies[txFunctionHash]; ok {
			return h(ctx, tx)
		}
	}
	return domain.ErrTxDoesntApply
}

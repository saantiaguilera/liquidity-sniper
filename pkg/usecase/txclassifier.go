package usecase

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/saantiaguilera/liquidity-AX-50/pkg/domain"
)

type (
	TransactionClassifier struct {
		mut *sync.Mutex // TODO: Check if necessary

		routerAddr string

		monitor    transactionClassifierMonitor
		strategies map[[4]byte]TransactionClassifierStrategy
	}

	transactionClassifierMonitor  func(ctx context.Context, tx *types.Transaction)
	TransactionClassifierStrategy func(ctx context.Context, tx *types.Transaction) error
)

func NewTransactionClassifier(
	raddr string,
	m transactionClassifierMonitor,
	s map[[4]byte]TransactionClassifierStrategy,
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

	//TODO: is calling sender() here needed?

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

package usecase

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type (
	TransactionClassifier struct {
		routerAddr string

		monitor    transactionClassifierMonitor
		strategies map[[4]byte]TransactionClassifierStrategy
	}

	transactionClassifierMonitor  func(ctx context.Context, tx *types.Transaction)
	TransactionClassifierStrategy func(ctx context.Context, tx *types.Transaction, pending bool) error
)

func NewTransactionClassifier(
	raddr string,
	m transactionClassifierMonitor,
	s map[[4]byte]TransactionClassifierStrategy,
) *TransactionClassifier {

	return &TransactionClassifier{
		routerAddr: raddr,
		monitor:    m,
		strategies: s,
	}
}

func (u *TransactionClassifier) Classify(ctx context.Context, tx *types.Transaction, pending bool) error {
	if tx.To() == nil {
		log.Trace("tx is a contract deploy: " + tx.Hash().String())
		return nil
	}

	u.monitor(ctx, tx)

	if tx.To().Hex() == u.routerAddr && len(tx.Data()) >= 4 {
		txFunctionHash := [4]byte{}
		copy(txFunctionHash[:], tx.Data()[:4])

		if h, ok := u.strategies[txFunctionHash]; ok {
			return h(ctx, tx, pending)
		}
		log.Debug("found contract call to provided router address but not to a method we are looking for: " + tx.Hash().String())
		return nil
	}

	log.Trace(fmt.Sprintf("tx %s doesn't apply", tx.Hash().String()))
	return nil
}

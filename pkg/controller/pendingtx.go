package controller

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type (
	// PendingTransaction controller allows us to consume pending transactions
	PendingTransaction struct {
		resolver pendingTransactionResolver
		handler  pendingTransactionHandler
	}

	pendingTransactionResolver interface {
		TransactionByHash(context.Context, common.Hash) (tx *types.Transaction, isPending bool, err error)
	}

	pendingTransactionHandler func(context.Context, *types.Transaction) error
)

func NewPendingTransaction(resolver pendingTransactionResolver, handler pendingTransactionHandler) *PendingTransaction {
	return &PendingTransaction{
		resolver: resolver,
		handler:  handler,
	}
}

func (c *PendingTransaction) Snipe(ctx context.Context, h common.Hash) error {
	log.Trace(fmt.Sprintf("new tx: %s", h.Hex()))

	// Get transaction object from hash by querying the client
	tx, pending, err := c.resolver.TransactionByHash(ctx, h)

	if err != nil {
		if err == ethereum.NotFound {
			log.Trace(fmt.Sprintf("tx not found: %s", h.Hex()))
			return nil // don't track. probably a failed tx
		}
		return fmt.Errorf("error getting tx %s by hash: %s", h.Hex(), err) // nothing to do.
	}

	// If tx is valid and still unconfirmed
	if pending {
		return c.handler(ctx, tx)
	}
	log.Warn(fmt.Sprintf("tx already confirmed: %s", h.Hex())) // we shouldn't be seeing txs confirmed, this means we are having a bottleneck against the read node
	return nil
}

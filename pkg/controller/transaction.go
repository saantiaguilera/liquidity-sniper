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
	// Transaction controller allows us to consume transactions
	Transaction struct {
		resolver transactionResolver
		handler  transactionHandler
	}

	transactionResolver interface {
		TransactionByHash(context.Context, common.Hash) (tx *types.Transaction, isPending bool, err error)
	}

	transactionHandler func(context.Context, *types.Transaction) error
)

func NewTransaction(resolver transactionResolver, handler transactionHandler) *Transaction {
	return &Transaction{
		resolver: resolver,
		handler:  handler,
	}
}

func (c *Transaction) Snipe(ctx context.Context, h common.Hash) error {
	log.Trace(fmt.Sprintf("new tx: %s", h.Hex()))

	// Get transaction object from hash by querying the client
	tx, pending, err := c.resolver.TransactionByHash(ctx, h)

	if err != nil {
		if err == ethereum.NotFound {
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

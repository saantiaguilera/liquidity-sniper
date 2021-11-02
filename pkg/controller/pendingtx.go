package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	pendingTransactionNotFoundMaxRetries = 2
	pendingTransactionNotFoundDelay      = 100 * time.Millisecond
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

	pendingTransactionHandler func(context.Context, *types.Transaction, bool) error

	pendingTransactionNotFoundKey struct{}
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

	if err == ethereum.NotFound {
		// pending txs may be dropped from the mempool after some time, hence we don't retry forever and have a deadline.
		tim, newCtx := c.newContextForRetries(ctx)
		if tim < pendingTransactionNotFoundMaxRetries {
			log.Debug(fmt.Sprintf("retrying not found tx: %s", h.Hex()))
			time.AfterFunc(pendingTransactionNotFoundDelay, func() {
				if err := c.Snipe(newCtx, h); err != nil {
					log.Error(err.Error())
				}
			})
		} else {
			log.Warn(fmt.Sprintf("tx not found: %s", h.Hex()))
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("error getting tx %s by hash: %s", h.Hex(), err) // nothing to do.
	}

	// If tx is valid and still unconfirmed
	if pending {
		return c.handler(ctx, tx, pending)
	}
	log.Warn(fmt.Sprintf("tx already confirmed: %s", h.Hex())) // we shouldn't be seeing txs confirmed, this means we are having a bottleneck against the read node
	return nil
}

func (c *PendingTransaction) newContextForRetries(ctx context.Context) (uint, context.Context) {
	var times uint = 0
	if v, ok := ctx.Value(pendingTransactionNotFoundKey{}).(uint); ok {
		times = v
	}
	return times, context.WithValue(ctx, pendingTransactionNotFoundKey{}, times+1)
}

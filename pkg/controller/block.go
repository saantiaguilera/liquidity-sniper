package controller

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	blockNotFoundRetryDuration = 200 * time.Millisecond
)

type (
	// Block controller allows us to consume pending transactions
	Block struct {
		resolver blockResolver
		handler  blockHandler
	}

	blockResolver interface {
		BlockByNumber(context.Context, *big.Int) (b *types.Block, err error)
	}

	blockHandler func(context.Context, *types.Transaction, bool) error
)

func NewBlock(resolver blockResolver, handler blockHandler) *Block {
	return &Block{
		resolver: resolver,
		handler:  handler,
	}
}

func (c *Block) Snipe(ctx context.Context, bn *big.Int) error {
	log.Debug(fmt.Sprintf("new block: %s", bn.String()))

	// Get block by querying the client
	b, err := c.resolver.BlockByNumber(ctx, bn)

	if err != nil {
		if err == ethereum.NotFound {
			// retry block forever with delay, all blocks should exist if they were added to the head.
			// probably because stream!=snipe node connections
			log.Warn(fmt.Sprintf("block %s not found: retrying", bn.String()))
			time.AfterFunc(blockNotFoundRetryDuration, func() {
				if err := c.Snipe(ctx, bn); err != nil {
					log.Error(err.Error())
				}
			})
			return nil
		}
		return fmt.Errorf("error getting block %s: %s", bn.String(), err) // nothing to do.
	}

	// Broadcast all txs in block
	var errs []error
	for _, tx := range b.Transactions() {
		if err := c.handler(ctx, tx, false); err != nil { // false = not pending. Since all txs are already confirmed at this point.
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		err := fmt.Errorf("error occured processing block %s", b.Number().String())
		for _, e := range errs {
			err = fmt.Errorf("%s: %s", err, e)
		}
		return err
	}
	log.Debug(fmt.Sprintf("block %s handled ok", b.Number().String()))
	return nil
}

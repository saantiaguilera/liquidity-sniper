package main

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/saantiaguilera/liquidity-AX-50/pkg/controller"
)

type (
	Engine struct {
		client *rpc.Client
		ctrl   *controller.Transaction
	}
)

func NewEngine(cl *rpc.Client, ctrl *controller.Transaction) *Engine {
	return &Engine{
		client: cl,
		ctrl:   ctrl,
	}
}

func (e *Engine) Run(ctx context.Context) {
	// Go channel to pipe data from client subscription
	ch := make(chan common.Hash)

	// Subscribe to receive one time events for new txs
	_, err := e.client.EthSubscribe(ctx, ch, "newPendingTransactions")

	if err != nil {
		panic(err)
	}

	// Eternal block for streaming until cancelled.
	for {
		select {
		case txHash := <-ch:
			go func() {
				defer recovery()
				if err := e.ctrl.Snipe(ctx, txHash); err != nil {
					log.Error(err.Error())
				}
			}()
		case <-ctx.Done():
			log.Info("ctx cancelled")
		}
	}
}

func recovery() {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
	}
}

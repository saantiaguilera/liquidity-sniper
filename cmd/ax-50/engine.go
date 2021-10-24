package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/saantiaguilera/liquidity-sniper/pkg/controller"
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
	var canc func()
	ctx, canc = context.WithCancel(ctx)

	// Go channel to pipe data from client subscription
	ch := make(chan common.Hash, workers)

	// Consume in workers the new txs
	wg := new(sync.WaitGroup)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(ctx context.Context, ch <-chan common.Hash, wg *sync.WaitGroup) {
			defer wg.Done()
			defer recovery(canc) // if a worker panics we stop everything
			e.consumeBlocking(ctx, ch)
		}(ctx, ch, wg)
	}

	// Subscribe to receive one time events for new txs
	e.subscribeSafe(ctx, ch, canc)

	// Block forever (or until the ctx gets cancelled / an error occurs)
	wg.Wait()
}

func (e *Engine) subscribeSafe(ctx context.Context, ch chan<- common.Hash, canc func()) {
	s, err := e.client.EthSubscribe(ctx, ch, "newPendingTransactions")
	if err != nil {
		panic(err)
	}
	go func() { // on error try subscribing again, if it panics we fail gracefully
		defer recovery(canc)
		var once sync.Once
		for range s.Err() {
			once.Do(func() {
				s.Unsubscribe()
				e.subscribeSafe(ctx, ch, canc)
			})
		}
	}()
}

func (e *Engine) consumeBlocking(ctx context.Context, ch <-chan common.Hash) {
	for {
		select {
		case txHash := <-ch:
			if err := e.ctrl.Snipe(ctx, txHash); err != nil {
				log.Error(err.Error())
			}
		case <-ctx.Done():
			return // break here.
		}
	}
}

func recovery(fn func()) {
	if err := recover(); err != nil {
		log.Error(fmt.Sprintf("panic recovered: %s %s", fmt.Errorf("%s", err), debug.Stack()))
		if fn != nil {
			fn()
		}
	}
}

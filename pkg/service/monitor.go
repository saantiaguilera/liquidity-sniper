package service

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

const (
	workers  = 100
	interval = 1 * time.Second
)

type (
	MonitorEngine struct {
		monitors []Monitor

		t *time.Ticker

		buf []entry
		mut *sync.Mutex
	}

	Monitor func(context.Context, *types.Transaction)

	entry struct {
		ctx context.Context
		tx  *types.Transaction
	}
)

func NewMonitorEngine(m ...Monitor) *MonitorEngine {
	e := &MonitorEngine{
		monitors: m,
		buf:      make([]entry, 0),
		mut:      new(sync.Mutex),
	}

	if len(m) > 0 {
		e.t = time.NewTicker(interval) // func-opts if this is OSS some day, Also expose Stop.
		go func(ch <-chan time.Time) {
			defer recovery()
			for range ch {
				e.monitorPendings()
			}
		}(e.t.C)
	}

	return e
}

func (e *MonitorEngine) Monitor(ctx context.Context, tx *types.Transaction) {
	if len(e.monitors) == 0 {
		return // noop
	}
	e.push(ctx, tx)
}

func (e *MonitorEngine) monitorPendings() {
	b := e.flush()

	if len(b) == 0 {
		return // nothing to do
	}

	w := workers
	if len(b) < workers {
		w = len(b)
	}

	ch := make(chan entry, len(b))
	for _, v := range b {
		ch <- v
	}
	close(ch)

	for i := 0; i < w; i++ {
		go func(ch <-chan entry) {
			defer recovery()
			for entr := range ch {
				for _, m := range e.monitors {
					m(entr.ctx, entr.tx)
				}
			}
		}(ch)
	}
}

func (e *MonitorEngine) push(ctx context.Context, tx *types.Transaction) {
	e.mut.Lock()
	defer e.mut.Unlock()
	e.buf = append(e.buf, entry{
		ctx: ctx,
		tx:  tx,
	})
}

func (e *MonitorEngine) flush() []entry {
	e.mut.Lock()
	defer e.mut.Unlock()
	buf := e.buf
	e.buf = make([]entry, 0)
	return buf
}

package worker

import (
	"context"
	"sync"

	"github.com/geiltonxavier/distributed-queue/internal/queue"
	"github.com/geiltonxavier/distributed-queue/internal/task"
)

// Handler processes a task. Returning an error is allowed; retry logic will be added in later steps.
type Handler func(ctx context.Context, t task.Task) error

// Pool runs a fixed number of workers consuming tasks from a broker.
type Pool struct {
	broker      queue.Broker
	workerCount int
	handler     Handler

	cancel func()
	wg     sync.WaitGroup
	once   sync.Once
}

// NewPool configures a worker pool with the given broker, worker count and handler.
func NewPool(broker queue.Broker, workerCount int, handler Handler) *Pool {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &Pool{
		broker:      broker,
		workerCount: workerCount,
		handler:     handler,
	}
}

// Start launches workers. It should be called once.
func (p *Pool) Start(ctx context.Context) {
	p.once.Do(func() {
		ctx, cancel := context.WithCancel(ctx)
		p.cancel = cancel
		for i := 0; i < p.workerCount; i++ {
			p.wg.Add(1)
			go p.runWorker(ctx)
		}
	})
}

// Stop signals workers to exit and waits for them.
func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *Pool) runWorker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		t, err := p.broker.Dequeue(ctx)
		if err != nil {
			// Exit on shutdown/closed broker or context cancellation.
			if err == queue.ErrQueueClosed || ctx.Err() != nil {
				return
			}
			continue
		}

		_ = p.handler(ctx, t)
	}
}

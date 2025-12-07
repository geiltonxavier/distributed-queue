package worker

import (
	"context"
	"sync"
	"time"

	"github.com/geiltonxavier/distributed-queue/internal/queue"
	"github.com/geiltonxavier/distributed-queue/internal/task"
)

// Handler processes a task. Returning an error triggers retry/backoff until max attempts are exhausted.
type Handler func(ctx context.Context, t task.Task) error

// Backoff defines how long to wait before the next attempt. attempt is 1-based (first retry is attempt=1).
type Backoff func(attempt int) time.Duration

// Option configures pool behavior.
type Option func(*Pool)

// Pool runs a fixed number of workers consuming tasks from a broker.
type Pool struct {
	broker      queue.Broker
	workerCount int
	handler     Handler

	maxAttempts int
	backoff     Backoff
	dlq         queue.DLQ

	cancel func()
	wg     sync.WaitGroup
	once   sync.Once
}

// NewPool configures a worker pool with the given broker, worker count and handler.
func NewPool(broker queue.Broker, workerCount int, handler Handler, opts ...Option) *Pool {
	if workerCount <= 0 {
		workerCount = 1
	}
	p := &Pool{
		broker:      broker,
		workerCount: workerCount,
		handler:     handler,
		maxAttempts: 3, // total attempts including the first one
		backoff:     ExponentialBackoff(50*time.Millisecond, time.Second),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.maxAttempts <= 0 {
		p.maxAttempts = 1
	}
	if p.backoff == nil {
		p.backoff = func(int) time.Duration { return 0 }
	}

	return p
}

// WithMaxAttempts sets the total attempts before sending to the DLQ.
func WithMaxAttempts(n int) Option {
	return func(p *Pool) {
		p.maxAttempts = n
	}
}

// WithBackoff overrides the backoff strategy between retries.
func WithBackoff(b Backoff) Option {
	return func(p *Pool) {
		p.backoff = b
	}
}

// WithDLQ configures a dead letter queue where failed tasks are sent after exhausting retries.
func WithDLQ(dlq queue.DLQ) Option {
	return func(p *Pool) {
		p.dlq = dlq
	}
}

// ExponentialBackoff returns a capped exponential backoff function.
func ExponentialBackoff(base, max time.Duration) Backoff {
	if base <= 0 {
		base = 10 * time.Millisecond
	}
	if max < base {
		max = base
	}
	return func(attempt int) time.Duration {
		if attempt < 1 {
			attempt = 1
		}
		d := base << (attempt - 1)
		if d > max {
			return max
		}
		return d
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

		p.processWithRetry(ctx, t)
	}
}

func (p *Pool) processWithRetry(ctx context.Context, t task.Task) {
	attempts := 0
	for {
		if ctx.Err() != nil {
			return
		}
		attempts++
		err := p.handler(ctx, t)
		if err == nil {
			return
		}
		if attempts >= p.maxAttempts {
			if p.dlq != nil {
				p.dlq.Publish(queue.Failure{
					Task:     t,
					Attempts: attempts,
					Err:      err,
					FailedAt: time.Now(),
				})
			}
			return
		}

		delay := p.backoff(attempts)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

package queue

import (
	"context"
	"errors"
	"sync"

	"github.com/geiltonxavier/distributed-queue/internal/task"
)

var (
	// ErrQueueClosed signals operations after broker shutdown.
	ErrQueueClosed = errors.New("queue closed")
)

// Broker describes minimal operations to enqueue and dequeue tasks.
type Broker interface {
	Enqueue(ctx context.Context, t task.Task) error
	Dequeue(ctx context.Context) (task.Task, error)
	Close()
}

// InMemoryBroker is a channel-backed broker for local development and tests.
type InMemoryBroker struct {
	capacity int
	ch       chan task.Task
	closed   chan struct{}
	once     sync.Once
}

// NewInMemoryBroker builds an in-memory broker with the given capacity.
func NewInMemoryBroker(capacity int) *InMemoryBroker {
	if capacity <= 0 {
		capacity = 1
	}
	return &InMemoryBroker{
		capacity: capacity,
		ch:       make(chan task.Task, capacity),
		closed:   make(chan struct{}),
	}
}

// Enqueue pushes a task into the broker respecting context cancellation.
func (b *InMemoryBroker) Enqueue(ctx context.Context, t task.Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.closed:
		return ErrQueueClosed
	default:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.closed:
		return ErrQueueClosed
	case b.ch <- t:
		return nil
	}
}

// Dequeue pops a task from the broker respecting context cancellation.
func (b *InMemoryBroker) Dequeue(ctx context.Context) (task.Task, error) {
	select {
	case <-ctx.Done():
		return task.Task{}, ctx.Err()
	case <-b.closed:
		return task.Task{}, ErrQueueClosed
	default:
	}
	select {
	case <-ctx.Done():
		return task.Task{}, ctx.Err()
	case <-b.closed:
		return task.Task{}, ErrQueueClosed
	case t, ok := <-b.ch:
		if !ok {
			return task.Task{}, ErrQueueClosed
		}
		return t, nil
	}
}

// Close stops the broker and drains pending consumers.
func (b *InMemoryBroker) Close() {
	b.once.Do(func() {
		close(b.closed)
		close(b.ch)
	})
}

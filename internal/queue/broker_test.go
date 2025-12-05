package queue

import (
	"context"
	"testing"
	"time"

	"github.com/geiltonxavier/distributed-queue/internal/task"
)

func TestEnqueueDequeue(t *testing.T) {
	broker := NewInMemoryBroker(2)
	defer broker.Close()

	ctx := context.Background()
	t1 := task.New("1", "email", []byte("hello"))
	t2 := task.New("2", "email", []byte("world"))

	if err := broker.Enqueue(ctx, t1); err != nil {
		t.Fatalf("enqueue t1: %v", err)
	}
	if err := broker.Enqueue(ctx, t2); err != nil {
		t.Fatalf("enqueue t2: %v", err)
	}

	got1, err := broker.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue 1: %v", err)
	}
	got2, err := broker.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue 2: %v", err)
	}

	if got1.ID != t1.ID || got2.ID != t2.ID {
		t.Fatalf("unexpected tasks: got %v and %v", got1.ID, got2.ID)
	}
}

func TestContextCancellation(t *testing.T) {
	broker := NewInMemoryBroker(1)
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := broker.Enqueue(ctx, task.New("1", "noop", nil)); err == nil {
		t.Fatalf("expected enqueue to respect context cancellation")
	}

	if _, err := broker.Dequeue(ctx); err == nil {
		t.Fatalf("expected dequeue to respect context cancellation")
	}
}

func TestCloseStopsOperations(t *testing.T) {
	broker := NewInMemoryBroker(1)
	defer broker.Close()

	if err := broker.Enqueue(context.Background(), task.New("1", "noop", nil)); err != nil {
		t.Fatalf("enqueue before close: %v", err)
	}

	broker.Close()

	if err := broker.Enqueue(context.Background(), task.New("2", "noop", nil)); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed after close, got %v", err)
	}

	// Dequeue should see closed state.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := broker.Dequeue(ctx); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed on dequeue after close, got %v", err)
	}
}

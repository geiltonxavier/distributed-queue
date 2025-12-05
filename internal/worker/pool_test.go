package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geiltonxavier/distributed-queue/internal/queue"
	"github.com/geiltonxavier/distributed-queue/internal/task"
)

func TestPoolProcessesTasks(t *testing.T) {
	broker := queue.NewInMemoryBroker(4)
	defer broker.Close()

	var processed atomic.Int32
	var mu sync.Mutex
	seen := make(map[string]struct{})

	handler := func(ctx context.Context, t task.Task) error {
		processed.Add(1)
		mu.Lock()
		seen[t.ID] = struct{}{}
		mu.Unlock()
		return nil
	}

	pool := NewPool(broker, 3, handler)
	pool.Start(context.Background())
	defer pool.Stop()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		id := time.Now().Format("150405.000") + "-" + string(rune('a'+i))
		if err := broker.Enqueue(ctx, task.New(id, "test", nil)); err != nil {
			t.Fatalf("enqueue task %d: %v", i, err)
		}
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	for {
		if processed.Load() == 5 {
			break
		}
		select {
		case <-waitCtx.Done():
			t.Fatalf("timeout waiting workers to process tasks")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 5 {
		t.Fatalf("expected 5 unique tasks processed, got %d", len(seen))
	}
}

func TestPoolStopsGracefully(t *testing.T) {
	broker := queue.NewInMemoryBroker(1)
	defer broker.Close()

	started := make(chan struct{})
	block := make(chan struct{})

	handler := func(ctx context.Context, t task.Task) error {
		close(started)
		<-block
		return nil
	}

	pool := NewPool(broker, 1, handler)
	pool.Start(context.Background())

	if err := broker.Enqueue(context.Background(), task.New("1", "test", nil)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	<-started

	stopDone := make(chan struct{})
	go func() {
		pool.Stop()
		close(stopDone)
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		// Unblock handler and expect Stop to return quickly.
		close(block)
	case <-stopDone:
		t.Fatalf("Stop returned while handler still blocked")
	}

	select {
	case <-stopDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Stop did not return after handler unblocked")
	}
}

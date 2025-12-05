package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/geiltonxavier/distributed-queue/internal/queue"
	"github.com/geiltonxavier/distributed-queue/internal/task"
	"github.com/geiltonxavier/distributed-queue/internal/worker"
)

// Simple demo CLI: spins up an in-memory broker, a worker pool, enqueues tasks and shows processing.
func main() {
	workers := envOrDefault("WORKERS", 3)
	taskCount := envOrDefault("TASKS", 10)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	broker := queue.NewInMemoryBroker(taskCount)
	defer broker.Close()

	pool := worker.NewPool(broker, workers, logHandler)
	pool.Start(ctx)
	defer pool.Stop()

	fmt.Printf("demo: starting with %d workers, %d tasks\n", workers, taskCount)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		enqueueTasks(ctx, broker, taskCount)
	}()

	wg.Wait()
	fmt.Println("demo: publisher done; waiting for workers to finish remaining tasks...")
	pool.Stop()
	fmt.Println("demo: shutdown complete")
}

func enqueueTasks(ctx context.Context, broker queue.Broker, count int) {
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("task-%02d", i)
		payload := []byte(fmt.Sprintf("payload-%d", i))
		t := task.New(id, "demo", payload)
		if err := broker.Enqueue(ctx, t); err != nil {
			fmt.Fprintf(os.Stderr, "enqueue error: %v\n", err)
			return
		}
		fmt.Printf("enqueued: %s\n", t.ID)
		time.Sleep(50 * time.Millisecond)
	}
}

func logHandler(ctx context.Context, t task.Task) error {
	start := time.Now()
	// Simulate variable work.
	processTime := time.Duration(50+rand.Intn(200)) * time.Millisecond
	time.Sleep(processTime)
	fmt.Printf("worker: processed %s in %s\n", t.ID, time.Since(start).Truncate(time.Millisecond))
	return nil
}

func envOrDefault(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

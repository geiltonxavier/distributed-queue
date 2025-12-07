package queue

import (
	"sync"
	"time"

	"github.com/geiltonxavier/distributed-queue/internal/task"
)

// Failure captures a task that could not be processed after retries.
type Failure struct {
	Task     task.Task
	Attempts int
	Err      error
	FailedAt time.Time
}

// DLQ receives failed tasks.
type DLQ interface {
	Publish(f Failure)
}

// InMemoryDLQ stores failures in memory for debugging/tests.
type InMemoryDLQ struct {
	mu       sync.Mutex
	failures []Failure
}

// NewInMemoryDLQ builds an in-memory DLQ.
func NewInMemoryDLQ() *InMemoryDLQ {
	return &InMemoryDLQ{}
}

// Publish records a failure.
func (d *InMemoryDLQ) Publish(f Failure) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failures = append(d.failures, f)
}

// Failures returns a copy of all failures recorded so far.
func (d *InMemoryDLQ) Failures() []Failure {
	d.mu.Lock()
	defer d.mu.Unlock()
	cp := make([]Failure, len(d.failures))
	copy(cp, d.failures)
	return cp
}

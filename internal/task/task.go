package task

import "time"

// Task represents a unit of work to be processed by a worker.
type Task struct {
	ID         string
	Type       string
	Payload    []byte
	EnqueuedAt time.Time
}

// New creates a task with timestamp filled.
func New(id, typ string, payload []byte) Task {
	return Task{
		ID:         id,
		Type:       typ,
		Payload:    payload,
		EnqueuedAt: time.Now(),
	}
}

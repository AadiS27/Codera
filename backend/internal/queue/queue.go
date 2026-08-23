package queue

import (
	"context"
	"errors"
)

var (
	ErrQueueFull   = errors.New("execution queue is full")
	ErrQueueClosed = errors.New("execution queue is closed")
)

// JobQueue defines the interface for an asynchronous job queue.
type JobQueue interface {
	// Enqueue adds a job ID to the queue.
	// Returns ErrQueueFull if the queue is at capacity.
	// Returns ErrQueueClosed if the queue has been shutdown.
	Enqueue(ctx context.Context, jobID string) error

	// Dequeue blocks until a job ID is available and returns it.
	// If the queue is closed, returns ErrQueueClosed.
	Dequeue(ctx context.Context) (string, error)

	// Close gracefully shuts down the queue, preventing new enqueues.
	Close()
}

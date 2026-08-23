package queue

import (
	"context"
	"errors"
)

var (
	ErrQueueFull   = errors.New("execution queue is full")
	ErrQueueClosed = errors.New("execution queue is closed")
)

// QueueMessage represents a job dispatched via the queue that requires an acknowledgement.
type QueueMessage struct {
	ID    string // E.g., the Redis Stream Message ID
	JobID string // The actual Execution Job ID
}

// JobQueue defines the interface for a distributed, acknowledged job queue.
type JobQueue interface {
	// Enqueue adds a job ID to the queue.
	Enqueue(ctx context.Context, jobID string) error

	// Consume blocks until a job is available and returns it.
	// consumerName is used to track ownership (e.g., in a Redis Consumer Group).
	Consume(ctx context.Context, consumerName string) (QueueMessage, error)

	// Ack acknowledges that the message has been durably handled.
	Ack(ctx context.Context, messageID string) error

	// Close gracefully shuts down the queue.
	Close()
}

package queue

import (
	"context"
	"sync/atomic"
)

type MemoryQueue struct {
	ch     chan string
	closed atomic.Bool
}

func NewMemoryQueue(capacity int64) *MemoryQueue {
	return &MemoryQueue{
		ch: make(chan string, capacity),
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, jobID string) error {
	if q.closed.Load() {
		return ErrQueueClosed
	}

	select {
	case q.ch <- jobID:
		return nil
	default:
		// Channel is full; do not block HTTP handler
		return ErrQueueFull
	}
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (string, error) {
	select {
	case jobID, ok := <-q.ch:
		if !ok {
			return "", ErrQueueClosed
		}
		return jobID, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (q *MemoryQueue) Close() {
	if q.closed.CompareAndSwap(false, true) {
		close(q.ch)
	}
}

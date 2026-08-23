package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client       *redis.Client
	stream       string
	group        string
	maxLen       int64
	closed       atomic.Bool
}

func NewRedisQueue(client *redis.Client, stream, group string, maxLen int64) *RedisQueue {
	return &RedisQueue{
		client: client,
		stream: stream,
		group:  group,
		maxLen: maxLen,
	}
}

// EnsureGroupExists tries to create the consumer group safely.
func (q *RedisQueue) EnsureGroupExists(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.stream, q.group, "0").Err()
	if err != nil {
		// Ignore BUSYGROUP (Group already exists) error
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			return nil
		}
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobID string) error {
	if q.closed.Load() {
		return ErrQueueClosed
	}

	err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.stream,
		MaxLen: q.maxLen,
		Values: map[string]interface{}{
			"job_id": jobID,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to enqueue to redis stream: %w", err)
	}
	return nil
}

func (q *RedisQueue) Consume(ctx context.Context, consumerName string) (QueueMessage, error) {
	for {
		if q.closed.Load() {
			return QueueMessage{}, ErrQueueClosed
		}

		// Blocking read from the stream
		streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    q.group,
			Consumer: consumerName,
			Streams:  []string{q.stream, ">"}, // ">" means new messages
			Count:    1,
			Block:    2 * time.Second, // Block for 2 seconds to allow checking for cancellation/close
		}).Result()

		if err != nil {
			if err == redis.Nil {
				// Block timeout, continue loop
				continue
			}
			// Context canceled?
			if ctx.Err() != nil {
				return QueueMessage{}, ctx.Err()
			}
			return QueueMessage{}, fmt.Errorf("redis XREADGROUP error: %w", err)
		}

		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			jobID, ok := msg.Values["job_id"].(string)
			if !ok {
				// Bad message format, maybe ACK it and move on?
				// For now, let's just log and continue, but we don't have a logger injected.
				continue
			}

			return QueueMessage{
				ID:    msg.ID,
				JobID: jobID,
			}, nil
		}
	}
}

func (q *RedisQueue) Ack(ctx context.Context, messageID string) error {
	return q.client.XAck(ctx, q.stream, q.group, messageID).Err()
}

func (q *RedisQueue) Close() {
	q.closed.Store(true)
}

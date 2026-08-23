package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func getPostgresPool(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://codera:codera_password@localhost:5432/codera_db?sslmode=disable")
	if err != nil {
		t.Skipf("Skipping DB test: unable to connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping DB test: unable to ping: %v", err)
	}
	return pool
}

func getRedisClient(t *testing.T) *redis.Client {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping Redis test: unable to ping: %v", err)
	}
	return client
}

func TestRedisQueueAndPostgresIntegration(t *testing.T) {
	pgPool := getPostgresPool(t)
	defer pgPool.Close()

	redisClient := getRedisClient(t)
	defer redisClient.Close()

	ctx := context.Background()
	store := jobs.NewPostgresJobStore(pgPool)
	
	streamName := "test-execution-jobs"
	groupName := "test-execution-workers"
	redisQueue := queue.NewRedisQueue(redisClient, streamName, groupName, 1000)

	// Clean up redis state before and after
	redisClient.Del(ctx, streamName)
	defer redisClient.Del(ctx, streamName)
	
	err := redisQueue.EnsureGroupExists(ctx)
	if err != nil {
		t.Fatalf("EnsureGroupExists failed: %v", err)
	}

	job := &jobs.ExecutionJob{
		ID:         "test_job_redis_1",
		Language:   "java",
		SourceCode: "code",
		Status:     jobs.StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	// 1. Store Job
	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer pgPool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	// 2. Enqueue Job
	if err := redisQueue.Enqueue(ctx, job.ID); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// 3. Consume Job
	consumeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	msg, err := redisQueue.Consume(consumeCtx, "worker-1")
	if err != nil {
		t.Fatalf("Consume failed: %v", err)
	}
	if msg.JobID != job.ID {
		t.Fatalf("Expected job ID %s, got %s", job.ID, msg.JobID)
	}

	// 4. Claim and Mark Running
	leaseDur := 10 * time.Second
	if err := store.Claim(ctx, msg.JobID, "worker-1", leaseDur); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if err := store.MarkRunning(ctx, msg.JobID, "worker-1"); err != nil {
		t.Fatalf("MarkRunning failed: %v", err)
	}

	// 5. Complete Job
	res := domain.ExecutionResult{
		Status: domain.StatusSuccess,
	}
	if err := store.Complete(ctx, msg.JobID, "worker-1", res); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// 6. Ack Message
	if err := redisQueue.Ack(ctx, msg.ID); err != nil {
		t.Fatalf("Ack failed: %v", err)
	}
}

func TestDuplicateDeliveryProtection(t *testing.T) {
	pgPool := getPostgresPool(t)
	defer pgPool.Close()

	ctx := context.Background()
	store := jobs.NewPostgresJobStore(pgPool)

	job := &jobs.ExecutionJob{
		ID:         "test_job_redis_dup_1",
		Language:   "java",
		SourceCode: "code",
		Status:     jobs.StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer pgPool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	// Simulate Worker A claiming it successfully
	if err := store.Claim(ctx, job.ID, "worker-A", 10*time.Second); err != nil {
		t.Fatalf("Worker A claim failed: %v", err)
	}
	if err := store.MarkRunning(ctx, job.ID, "worker-A"); err != nil {
		t.Fatalf("Worker A MarkRunning failed: %v", err)
	}

	// Simulate Worker B receiving a duplicate delivery of the SAME job and trying to claim it
	err := store.Claim(ctx, job.ID, "worker-B", 10*time.Second)
	if err != jobs.ErrInvalidState {
		t.Fatalf("Worker B should have failed with ErrInvalidState, got %v", err)
	}
}

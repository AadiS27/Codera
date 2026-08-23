package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
	"github.com/codera/code-executor/internal/recovery"
)

func TestAbandonedJobRecovery(t *testing.T) {
	pgPool := getPostgresPool(t)
	defer pgPool.Close()

	redisClient := getRedisClient(t)
	defer redisClient.Close()

	ctx := context.Background()
	store := jobs.NewPostgresJobStore(pgPool)
	redisQueue := queue.NewRedisQueue(redisClient, "test-stream-recovery", "test-group-recovery", 100)
	
	_ = redisQueue.EnsureGroupExists(ctx)

	job := &jobs.ExecutionJob{
		ID:         "test_job_abandoned",
		Language:   "python",
		SourceCode: "print('hello')",
		Status:     jobs.StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	defer pgPool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	// Claim it with a negative lease (already expired)
	if err := store.Claim(ctx, job.ID, "worker-crashing", -1*time.Second); err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := recovery.NewService(pgPool, logger, redisQueue, 5*time.Second, 20*time.Second)
	
	// Run recovery directly
	svc.RecoverAbandonedJobs(ctx)

	// Verify state is QUEUED and NextRetryAt is set
	j, _ := store.Get(ctx, job.ID)
	if j.Status != jobs.StatusQueued {
		t.Fatalf("Expected job to be QUEUED, got %s", j.Status)
	}
	if j.WorkerID != nil {
		t.Fatalf("Expected WorkerID to be nil")
	}
	if j.NextRetryAt == nil {
		t.Fatalf("Expected NextRetryAt to be set")
	}

	// Now dispatch it
	svc.DispatchDelayedJobs(ctx)

	// Consume it from Redis to prove it was redispatched
	msg, err := redisQueue.Consume(ctx, "worker-new")
	if err != nil {
		t.Fatalf("Failed to consume redispatched job: %v", err)
	}
	if msg.JobID != job.ID {
		t.Fatalf("Expected to consume %s, got %s", job.ID, msg.JobID)
	}
}

func TestMaxAttemptsDeadLettered(t *testing.T) {
	pgPool := getPostgresPool(t)
	defer pgPool.Close()

	ctx := context.Background()
	store := jobs.NewPostgresJobStore(pgPool)

	job := &jobs.ExecutionJob{
		ID:          "test_job_max_attempts",
		Language:    "python",
		SourceCode:  "print('fail')",
		Status:      jobs.StatusQueued,
		CreatedAt:   time.Now().UTC(),
		MaxAttempts: 1, // Only 1 attempt allowed
	}

	if err := store.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	defer pgPool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	// Claim it
	if err := store.Claim(ctx, job.ID, "worker-1", 10*time.Second); err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}

	// Fail it
	if err := store.FailJob(ctx, job.ID, "worker-1", "Docker unavailable", 1*time.Second); err != nil {
		t.Fatalf("Failed to FailJob: %v", err)
	}

	// It should now be DEAD_LETTERED because AttemptCount (1) >= MaxAttempts (1)
	j, _ := store.Get(ctx, job.ID)
	if j.Status != jobs.StatusDeadLettered {
		t.Fatalf("Expected DEAD_LETTERED, got %s", j.Status)
	}
	if j.DeadLetteredAt == nil {
		t.Fatalf("Expected DeadLetteredAt to be set")
	}
}

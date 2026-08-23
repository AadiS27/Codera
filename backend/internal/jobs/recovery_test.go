package jobs

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/queue"
)

func TestStartupRecovery(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	store := NewPostgresJobStore(pool)
	q := queue.NewMemoryQueue(10)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	jobRunning := &ExecutionJob{
		ID:         "recovery_job_running",
		Language:   "java",
		SourceCode: "code",
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	jobQueued := &ExecutionJob{
		ID:         "recovery_job_queued",
		Language:   "java",
		SourceCode: "code",
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	_ = store.Create(ctx, jobRunning)
	_ = store.Create(ctx, jobQueued)
	defer func() {
		pool.Exec(ctx, "DELETE FROM executions WHERE id IN ($1, $2)", jobRunning.ID, jobQueued.ID)
	}()

	_ = store.MarkRunning(ctx, jobRunning.ID)

	// Run recovery
	if err := RunStartupRecovery(ctx, store, q, logger); err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}

	// 1. Verify RUNNING job was marked as INTERNAL_ERROR
	savedRunning, _ := store.Get(ctx, jobRunning.ID)
	if savedRunning.Status != StatusCompleted {
		t.Errorf("Expected running job to be completed, got %s", savedRunning.Status)
	}
	if savedRunning.Result == nil || savedRunning.Result.Status != domain.StatusRuntimeError {
		t.Errorf("Expected internal error status, got %+v", savedRunning.Result)
	}

	// 2. Verify QUEUED job was enqueued
	jobID, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}
	if jobID != jobQueued.ID {
		t.Errorf("Expected to dequeue %s, got %s", jobQueued.ID, jobID)
	}
}

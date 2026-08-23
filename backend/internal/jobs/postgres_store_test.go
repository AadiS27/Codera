package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// In a real project, we would use testcontainers-go or a dedicated test DB.
// Since we are validating local DB integration, we assume a local DB is running.
func getTestPool(t *testing.T) *pgxpool.Pool {
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

func TestPostgresJobStore_CreateAndGet(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	store := NewPostgresJobStore(pool)
	ctx := context.Background()

	job := &ExecutionJob{
		ID:         "test_job_1",
		Language:   "java",
		SourceCode: "public class Main {}",
		Input:      "test",
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	err := store.Create(ctx, job)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	saved, err := store.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if saved.Language != job.Language || saved.Status != job.Status {
		t.Errorf("Saved job mismatch: got %+v", saved)
	}
}

func TestPostgresJobStore_AtomicTransitions(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	store := NewPostgresJobStore(pool)
	ctx := context.Background()

	job := &ExecutionJob{
		ID:         "test_job_2",
		Language:   "java",
		SourceCode: "code",
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	_ = store.Create(ctx, job)
	defer pool.Exec(ctx, "DELETE FROM executions WHERE id = $1", job.ID)

	// Transition QUEUED -> RUNNING
	if err := store.MarkRunning(ctx, job.ID); err != nil {
		t.Fatalf("MarkRunning failed: %v", err)
	}

	// Transition QUEUED -> RUNNING AGAIN should fail
	if err := store.MarkRunning(ctx, job.ID); err != ErrInvalidState {
		t.Fatalf("Expected ErrInvalidState, got %v", err)
	}

	// Transition RUNNING -> COMPLETED
	res := domain.ExecutionResult{
		Status:   domain.StatusSuccess,
		Stdout:   "hello",
		ExitCode: 0,
	}
	if err := store.Complete(ctx, job.ID, res); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Transition COMPLETED -> RUNNING should fail
	if err := store.MarkRunning(ctx, job.ID); err != ErrInvalidState {
		t.Fatalf("Expected ErrInvalidState for COMPLETED->RUNNING, got %v", err)
	}
}

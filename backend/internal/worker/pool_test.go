package worker

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
)

// mockExecutor fulfills execution.Executor
type mockExecutor struct {
	mu           sync.Mutex
	active       int
	maxActive    int
	executeDelay time.Duration
}

func (m *mockExecutor) Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	m.mu.Lock()
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.active--
		m.mu.Unlock()
	}()

	time.Sleep(m.executeDelay)

	return domain.ExecutionResult{
		Status:   domain.StatusSuccess,
		ExitCode: 0,
	}, nil
}

func TestWorkerConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := jobs.NewMemoryJobStore()
	q := queue.NewMemoryQueue(100)

	mockExec := &mockExecutor{
		executeDelay: 200 * time.Millisecond,
	}

	// Create a service with the mock executor
	execService := execution.NewServiceForTesting(mockExec)

	workers := int64(2)
	pool := NewPool(logger, q, store, execService, workers)
	
	pool.Start(context.Background())
	defer pool.Stop(context.Background())

	// Submit 10 jobs
	for i := 0; i < 10; i++ {
		job := &jobs.ExecutionJob{
			ID:         "job_" + string(rune('A'+i)),
			Language:   "java",
			SourceCode: "code",
			Status:     jobs.StatusQueued,
		}
		store.Create(context.Background(), job)
		q.Enqueue(context.Background(), job.ID)
	}

	// Wait for them all to finish
	time.Sleep(2 * time.Second)

	mockExec.mu.Lock()
	maxConcurrent := mockExec.maxActive
	mockExec.mu.Unlock()

	if maxConcurrent > int(workers) {
		t.Errorf("expected max %d concurrent workers, got %d", workers, maxConcurrent)
	}
}

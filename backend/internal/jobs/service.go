package jobs

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/queue"
	"github.com/oklog/ulid/v2"
)

type Service struct {
	store    JobStore
	queue    queue.JobQueue
	registry language.Registry
}

func NewService(store JobStore, q queue.JobQueue, registry language.Registry) *Service {
	return &Service{
		store:    store,
		queue:    q,
		registry: registry,
	}
}

// CreateExecution validates the request, generates an ID, stores the job, and enqueues it.
func (s *Service) CreateExecution(ctx context.Context, req domain.ExecutionRequest) (*ExecutionJob, error) {
	// Simple validation
	if req.Language == "" || req.SourceCode == "" {
		return nil, fmt.Errorf("language and source_code are required")
	}

	// Validate language against registry
	if _, err := s.registry.Get(req.Language); err != nil {
		return nil, fmt.Errorf("unsupported language: %v", req.Language)
	}

	// Generate ULID
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	jobID := "exec_" + id.String()

	job := &ExecutionJob{
		ID:         jobID,
		Language:   req.Language,
		SourceCode: req.SourceCode,
		Input:      req.Input,
		Status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}

	// 1. Store the job first
	if err := s.store.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to store job: %w", err)
	}

	// 2. Enqueue the job ID
	if err := s.queue.Enqueue(ctx, job.ID); err != nil {
		return nil, err // Could be ErrQueueFull or ErrQueueClosed
	}

	return job.Clone(), nil
}

// GetExecution fetches a job by ID
func (s *Service) GetExecution(ctx context.Context, id string) (*ExecutionJob, error) {
	return s.store.Get(ctx, id)
}

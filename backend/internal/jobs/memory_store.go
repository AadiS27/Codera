package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/codera/code-executor/internal/domain"
)

type MemoryJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*ExecutionJob
}

func NewMemoryJobStore() *MemoryJobStore {
	return &MemoryJobStore{
		jobs: make(map[string]*ExecutionJob),
	}
}

func (s *MemoryJobStore) Create(ctx context.Context, job *ExecutionJob) error {
	if job.Status != StatusQueued {
		return ErrInvalidState
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = job.Clone() // store a copy
	return nil
}

func (s *MemoryJobStore) Get(ctx context.Context, id string) (*ExecutionJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, ErrJobNotFound
	}
	
	return job.Clone(), nil // return a copy to prevent mutation outside lock
}

func (s *MemoryJobStore) MarkRunning(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	if job.Status != StatusQueued {
		return ErrInvalidState
	}

	job.Status = StatusRunning
	now := time.Now().UTC()
	job.StartedAt = &now
	
	return nil
}

func (s *MemoryJobStore) Complete(ctx context.Context, id string, result domain.ExecutionResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	if job.Status != StatusRunning {
		return ErrInvalidState
	}

	job.Status = StatusCompleted
	now := time.Now().UTC()
	job.CompletedAt = &now
	job.Result = &result
	
	return nil
}

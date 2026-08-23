package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/codera/code-executor/internal/domain"
)

var (
	ErrJobNotFound        = errors.New("job not found")
	ErrInvalidState       = errors.New("invalid job state transition")
)

// JobStore defines the persistence layer for execution jobs.
// It acts as a state machine, strictly enforcing state transitions.
type JobStore interface {
	// Create persists a new job in the store. Job must be QUEUED.
	Create(ctx context.Context, job *ExecutionJob) error
	
	// Get retrieves a snapshot of the job by ID.
	Get(ctx context.Context, id string) (*ExecutionJob, error)
	
	// MarkRunning atomically transitions a job from QUEUED to RUNNING.
	// Fails if the job is not in QUEUED state.
	MarkRunning(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error
	
	// Complete atomically transitions a job from RUNNING to COMPLETED and stores the result.
	// Fails if the job is not in RUNNING state.
	Complete(ctx context.Context, id string, workerID string, result domain.ExecutionResult) error
}

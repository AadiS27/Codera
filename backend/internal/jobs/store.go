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
	
	// Claim atomically transitions a job from QUEUED to CLAIMED and assigns ownership.
	Claim(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error

	// MarkRunning atomically transitions a job from CLAIMED to RUNNING.
	MarkRunning(ctx context.Context, id string, workerID string) error
	
	// Complete atomically transitions a job from RUNNING to COMPLETED and stores the result.
	// Fails if the job is not owned by the worker.
	Complete(ctx context.Context, id string, workerID string, results []domain.ExecutionResult) error

	// FailJob handles a retryable or permanent platform failure, moving the job to QUEUED or DEAD_LETTERED.
	FailJob(ctx context.Context, id string, workerID string, errMsg string, backoff time.Duration) error
	
	// RenewLease extends the lease for a CLAIMED or RUNNING job owned by a specific worker.
	RenewLease(ctx context.Context, id string, workerID string, leaseDuration time.Duration) error
}

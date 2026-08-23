package jobs

import (
	"time"

	"github.com/codera/code-executor/internal/domain"
)

// JobStatus represents the lifecycle state of a job in the queue/worker system.
type JobStatus string

const (
	StatusQueued       JobStatus = "QUEUED"
	StatusClaimed      JobStatus = "CLAIMED"
	StatusRunning      JobStatus = "RUNNING"
	StatusCompleted    JobStatus = "COMPLETED"
	StatusDeadLettered JobStatus = "DEAD_LETTERED"
)

// ExecutionJob represents an async execution task in the system.
type ExecutionJob struct {
	ID         string
	Language   domain.Language
	SourceCode string
	Input      string
	
	Status JobStatus
	Result *domain.ExecutionResult // nil until Status == StatusCompleted

	CreatedAt   time.Time
	ClaimedAt   *time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	
	WorkerID         *string
	AttemptCount     int
	MaxAttempts      int
	NextRetryAt      *time.Time
	LeaseExpiresAt   *time.Time
	LastError        *string
	DeadLetteredAt   *time.Time
	LastDispatchedAt *time.Time
}

// Clone returns a deep copy of the job to prevent data races and encapsulate state
func (j *ExecutionJob) Clone() *ExecutionJob {
	if j == nil {
		return nil
	}

	clone := &ExecutionJob{
		ID:         j.ID,
		Language:   j.Language,
		SourceCode: j.SourceCode,
		Input:      j.Input,
		Status:     j.Status,
		CreatedAt:  j.CreatedAt,
	}

	if j.StartedAt != nil {
		t := *j.StartedAt
		clone.StartedAt = &t
	}
	if j.CompletedAt != nil {
		t := *j.CompletedAt
		clone.CompletedAt = &t
	}
	if j.Result != nil {
		r := *j.Result
		clone.Result = &r
	}
	if j.ClaimedAt != nil {
		t := *j.ClaimedAt
		clone.ClaimedAt = &t
	}
	if j.WorkerID != nil {
		s := *j.WorkerID
		clone.WorkerID = &s
	}
	clone.AttemptCount = j.AttemptCount
	clone.MaxAttempts = j.MaxAttempts
	if j.NextRetryAt != nil {
		t := *j.NextRetryAt
		clone.NextRetryAt = &t
	}
	if j.LeaseExpiresAt != nil {
		t := *j.LeaseExpiresAt
		clone.LeaseExpiresAt = &t
	}
	if j.LastError != nil {
		s := *j.LastError
		clone.LastError = &s
	}
	if j.DeadLetteredAt != nil {
		t := *j.DeadLetteredAt
		clone.DeadLetteredAt = &t
	}
	if j.LastDispatchedAt != nil {
		t := *j.LastDispatchedAt
		clone.LastDispatchedAt = &t
	}

	return clone
}

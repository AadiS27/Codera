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
	ID         string          `json:"id"`
	Language   domain.Language `json:"language"`
	SourceCode string          `json:"source_code"`
	Inputs     []string        `json:"inputs"`
	
	Status JobStatus           `json:"status"`
	Results []domain.ExecutionResult `json:"results"` // empty until Status == StatusCompleted

	CreatedAt   time.Time      `json:"created_at"`
	ClaimedAt   *time.Time     `json:"claimed_at"`
	StartedAt   *time.Time     `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	
	WorkerID         *string   `json:"worker_id"`
	AttemptCount     int       `json:"attempt_count"`
	MaxAttempts      int       `json:"max_attempts"`
	NextRetryAt      *time.Time `json:"next_retry_at"`
	LeaseExpiresAt   *time.Time `json:"lease_expires_at"`
	LastError        *string   `json:"last_error"`
	DeadLetteredAt   *time.Time `json:"dead_lettered_at"`
	LastDispatchedAt *time.Time `json:"last_dispatched_at"`
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
		Inputs:     append([]string(nil), j.Inputs...),
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
	if j.Results != nil {
		clone.Results = append([]domain.ExecutionResult(nil), j.Results...)
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

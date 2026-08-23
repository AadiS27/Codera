package jobs

import (
	"time"

	"github.com/codera/code-executor/internal/domain"
)

// JobStatus represents the lifecycle state of a job in the queue/worker system.
type JobStatus string

const (
	StatusQueued    JobStatus = "QUEUED"
	StatusRunning   JobStatus = "RUNNING"
	StatusCompleted JobStatus = "COMPLETED"
)

// ExecutionJob represents an async execution task in the system.
type ExecutionJob struct {
	ID         string
	Language   string
	SourceCode string
	Input      string
	
	Status JobStatus
	Result *domain.ExecutionResult // nil until Status == StatusCompleted

	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
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

	return clone
}

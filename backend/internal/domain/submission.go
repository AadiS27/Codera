package domain

import "time"

type SubmissionStatus string

const (
	SubmissionStatusQueued    SubmissionStatus = "QUEUED"
	SubmissionStatusRunning   SubmissionStatus = "RUNNING"
	SubmissionStatusCompleted SubmissionStatus = "COMPLETED"
	SubmissionStatusFailed    SubmissionStatus = "FAILED_RETRYABLE"
)

type Verdict string

const (
	VerdictAccepted            Verdict = "ACCEPTED"
	VerdictWrongAnswer         Verdict = "WRONG_ANSWER"
	VerdictTimeLimitExceeded   Verdict = "TIME_LIMIT_EXCEEDED"
	VerdictMemoryLimitExceeded Verdict = "MEMORY_LIMIT_EXCEEDED"
	VerdictRuntimeError        Verdict = "RUNTIME_ERROR"
	VerdictCompilationError    Verdict = "COMPILATION_ERROR"
	VerdictOutputLimitExceeded Verdict = "OUTPUT_LIMIT_EXCEEDED"
	VerdictInternalError       Verdict = "INTERNAL_ERROR"
	VerdictPending             Verdict = "PENDING"
)

type Submission struct {
	ID             string
	UserID         string
	ProblemID      string
	Language       Language
	SourceCode     string
	
	Status         SubmissionStatus
	Verdict        Verdict
	
	ExecutionTimeMs int
	MemoryUsedBytes int64
	
	PassedTestCases int
	TotalTestCases  int
	
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

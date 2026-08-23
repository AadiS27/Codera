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
	ID             string           `json:"id"`
	UserID         string           `json:"user_id"`
	ProblemID      string           `json:"problem_id"`
	Language       Language         `json:"language"`
	SourceCode     string           `json:"source_code"`
	
	Status         SubmissionStatus `json:"status"`
	Verdict        Verdict          `json:"verdict"`
	
	ExecutionTimeMs int             `json:"execution_time_ms"`
	MemoryUsedBytes int64           `json:"memory_used_bytes"`
	
	PassedTestCases int               `json:"passed_test_cases"`
	TotalTestCases  int               `json:"total_test_cases"`
	AITimeComplexity  string          `json:"ai_time_complexity,omitempty"`
	AISpaceComplexity string          `json:"ai_space_complexity,omitempty"`
	AIFeedback        string          `json:"ai_feedback,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
}

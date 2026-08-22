package domain

type ExecutionStatus string

const (
	StatusSuccess             ExecutionStatus = "SUCCESS"
	StatusCompilationError    ExecutionStatus = "COMPILATION_ERROR"
	StatusCompilationTimeout  ExecutionStatus = "COMPILATION_TIMEOUT"
	StatusRuntimeError        ExecutionStatus = "RUNTIME_ERROR"
	StatusTimeLimitExceeded   ExecutionStatus = "TIME_LIMIT_EXCEEDED"
	StatusOutputLimitExceeded ExecutionStatus = "OUTPUT_LIMIT_EXCEEDED"

	// Future phases:
	StatusQueued  ExecutionStatus = "QUEUED"
	StatusRunning ExecutionStatus = "RUNNING"
)

type ExecutionRequest struct {
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
	Input      string `json:"input"`
}

type ExecutionResult struct {
	Status   ExecutionStatus `json:"status"`
	Stdout   string          `json:"stdout"`
	Stderr   string          `json:"stderr"`
	ExitCode int             `json:"exit_code"`
}

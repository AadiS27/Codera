package domain

type Language string

const (
	LanguageJava   Language = "java"
	LanguagePython Language = "python"
	LanguageGo     Language = "go"
	LanguageCpp    Language = "cpp"
)
type ExecutionStatus string

const (
	StatusSuccess             ExecutionStatus = "SUCCESS"
	StatusCompilationError    ExecutionStatus = "COMPILATION_ERROR"
	StatusCompilationTimeout  ExecutionStatus = "COMPILATION_TIMEOUT"
	StatusRuntimeError        ExecutionStatus = "RUNTIME_ERROR"
	StatusTimeLimitExceeded   ExecutionStatus = "TIME_LIMIT_EXCEEDED"
	StatusMemoryLimitExceeded ExecutionStatus = "MEMORY_LIMIT_EXCEEDED"
	StatusOutputLimitExceeded ExecutionStatus = "OUTPUT_LIMIT_EXCEEDED"
	StatusInternalError       ExecutionStatus = "INTERNAL_ERROR"

	// API states (mapped from job status)
	StatusQueued      ExecutionStatus = "QUEUED"
	StatusRunning     ExecutionStatus = "RUNNING"
	StatusDeadLetter  ExecutionStatus = "DEAD_LETTERED"
)

type ExecutionRequest struct {
	Language   Language `json:"language"`
	SourceCode string   `json:"source_code"`
	Input      string   `json:"input"`
}

type ExecutionResult struct {
	Status   ExecutionStatus `json:"status"`
	Stdout   string          `json:"stdout"`
	Stderr   string          `json:"stderr"`
	ExitCode int             `json:"exit_code"`
}

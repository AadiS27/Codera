package sandbox

import (
	"context"
	"io"
	"time"
)

// ExecOptions defines the options for a single command execution inside the sandbox
type ExecOptions struct {
	Command     string
	Args        []string
	Stdin       io.Reader
	StdoutLimit int64
	StderrLimit int64
	Timeout     time.Duration
}

// ExecResult contains the result of a single command execution
type ExecResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Timeout     bool
	OutputLimit bool
	Error       error
}

// Runtime defines the abstraction for a secure execution environment (e.g., Docker)
type Runtime interface {
	// StartContainer creates and starts an isolated container with the workspace mounted
	StartContainer(ctx context.Context, workspace string) (string, error)

	// Exec runs a command synchronously inside the specified container
	Exec(ctx context.Context, containerID string, opts ExecOptions) ExecResult

	// DestroyContainer forcefully removes the container
	DestroyContainer(containerID string) error
}

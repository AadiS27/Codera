package execution

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type RunOptions struct {
	Ctx         context.Context
	Command     string
	Args        []string
	Dir         string
	Stdin       io.Reader
	StdoutLimit int64
	StderrLimit int64
	Timeout     time.Duration
}

type ProcessResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Timeout     bool
	OutputLimit bool
	Error       error
}

func RunProcess(opts RunOptions) ProcessResult {
	// Create cancellation context for this specific run
	ctx, cancel := context.WithCancelCause(opts.Ctx)
	defer cancel(nil)

	// We apply a deadline manually via goroutine below to easily capture the reason,
	// or we can use context.WithTimeout. context.WithTimeout doesn't let us easily
	// differentiate between OutputLimit cancellation vs Timeout, so we'll use a timer.

	cmd := exec.CommandContext(ctx, opts.Command, opts.Args...)
	cmd.Dir = opts.Dir
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	// Attach OS specific attributes (e.g. process group)
	setSysProcAttr(cmd)

	outWriter := NewBoundedWriter(opts.StdoutLimit, cancel)
	errWriter := NewBoundedWriter(opts.StderrLimit, cancel)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	// 1. Start process
	err := cmd.Start()
	if err != nil {
		return ProcessResult{Error: err}
	}

	// We need to wait for process, timeout, or output limit cancellation.
	// Since cmd.Wait() blocks, we do it in a goroutine.
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
	}()

	timer := time.NewTimer(opts.Timeout)
	defer timer.Stop()

	var exitErr error
	var timeoutExceeded bool

	select {
	case <-timer.C:
		timeoutExceeded = true
		// Kill process group
		killProcess(cmd)
		// Wait for goroutine to finish (with killed error)
		<-errChan

	case <-ctx.Done():
		// Could be output limit exceeded (via cause) or parent ctx cancelled
		killProcess(cmd)
		<-errChan

	case err := <-errChan:
		exitErr = err
	}

	outputLimitExceeded := outWriter.Exceeded() || errWriter.Exceeded()

	// If context was cancelled by BoundedWriter, ensure we record it.
	if cause := context.Cause(ctx); errors.Is(cause, ErrOutputLimitExceeded) {
		outputLimitExceeded = true
	}

	exitCode := 0
	if exitErr != nil {
		exitCode = 1
		var exerr *exec.ExitError
		if errors.As(exitErr, &exerr) {
			exitCode = exerr.ExitCode()
		}
	}

	return ProcessResult{
		Stdout:      outWriter.String(),
		Stderr:      errWriter.String(),
		ExitCode:    exitCode,
		Timeout:     timeoutExceeded,
		OutputLimit: outputLimitExceeded,
		Error:       exitErr, // Preserve underlying error for internal platform failure check
	}
}

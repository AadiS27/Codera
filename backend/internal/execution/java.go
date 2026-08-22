package execution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
)

type JavaExecutor struct {
	config *config.Config
}

func NewJavaExecutor(cfg *config.Config) *JavaExecutor {
	return &JavaExecutor{config: cfg}
}

func isPlatformError(err error) bool {
	if err == nil {
		return false
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false // Process ran but failed, this is a user error
	}
	return true
}

func (j *JavaExecutor) Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	// Step 1: Create a unique temporary directory
	workspace, err := os.MkdirTemp("", "code-executor-java-*")
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workspace) // Step 7: Cleanup always

	// Step 2: Write source code
	sourcePath := filepath.Join(workspace, "Main.java")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0644); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to write source file: %w", err)
	}

	// Step 3: Compile
	compileOpts := RunOptions{
		Ctx:         ctx,
		Command:     "javac",
		Args:        []string{"-source", "1.8", "-target", "1.8", "Main.java"},
		Dir:         workspace,
		StdoutLimit: j.config.MaxStdoutBytes,
		StderrLimit: j.config.MaxStderrBytes,
		Timeout:     j.config.CompileTimeout,
	}

	compileRes := RunProcess(compileOpts)
	if isPlatformError(compileRes.Error) {
		return domain.ExecutionResult{}, fmt.Errorf("failed to compile process (platform): %w", compileRes.Error)
	}

	if compileRes.Timeout {
		return domain.ExecutionResult{
			Status:   domain.StatusCompilationTimeout,
			Stdout:   compileRes.Stdout,
			Stderr:   compileRes.Stderr,
			ExitCode: compileRes.ExitCode,
		}, nil
	}

	if compileRes.OutputLimit {
		return domain.ExecutionResult{
			Status:   domain.StatusOutputLimitExceeded,
			Stdout:   compileRes.Stdout,
			Stderr:   compileRes.Stderr,
			ExitCode: compileRes.ExitCode,
		}, nil
	}

	if compileRes.ExitCode != 0 {
		return domain.ExecutionResult{
			Status:   domain.StatusCompilationError,
			Stdout:   compileRes.Stdout,
			Stderr:   compileRes.Stderr,
			ExitCode: compileRes.ExitCode,
		}, nil
	}

	// Step 5: Run
	var stdinReader io.Reader
	if req.Input != "" {
		stdinReader = strings.NewReader(req.Input)
	}

	runOpts := RunOptions{
		Ctx:         ctx,
		Command:     "java",
		Args:        []string{"Main"},
		Dir:         workspace,
		Stdin:       stdinReader,
		StdoutLimit: j.config.MaxStdoutBytes,
		StderrLimit: j.config.MaxStderrBytes,
		Timeout:     j.config.RunTimeout,
	}

	runRes := RunProcess(runOpts)
	if isPlatformError(runRes.Error) {
		return domain.ExecutionResult{}, fmt.Errorf("failed to run process (platform): %w", runRes.Error)
	}

	if runRes.Timeout {
		return domain.ExecutionResult{
			Status:   domain.StatusTimeLimitExceeded,
			Stdout:   runRes.Stdout,
			Stderr:   runRes.Stderr,
			ExitCode: runRes.ExitCode,
		}, nil
	}

	if runRes.OutputLimit {
		return domain.ExecutionResult{
			Status:   domain.StatusOutputLimitExceeded,
			Stdout:   runRes.Stdout,
			Stderr:   runRes.Stderr,
			ExitCode: runRes.ExitCode,
		}, nil
	}

	if runRes.ExitCode != 0 {
		return domain.ExecutionResult{
			Status:   domain.StatusRuntimeError,
			Stdout:   runRes.Stdout,
			Stderr:   runRes.Stderr,
			ExitCode: runRes.ExitCode,
		}, nil
	}

	// Step 6: Classify result as SUCCESS
	return domain.ExecutionResult{
		Status:   domain.StatusSuccess,
		Stdout:   runRes.Stdout,
		Stderr:   runRes.Stderr,
		ExitCode: 0,
	}, nil
}

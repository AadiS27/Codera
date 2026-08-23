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
	"github.com/codera/code-executor/internal/sandbox"
)

type JavaExecutor struct {
	config  *config.Config
	sandbox sandbox.Runtime
}

func NewJavaExecutor(cfg *config.Config, sb sandbox.Runtime) *JavaExecutor {
	return &JavaExecutor{config: cfg, sandbox: sb}
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
	// Step 1: Create a unique temporary directory on host
	workspace, err := os.MkdirTemp("", "code-executor-java-*")
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workspace) // Step 7: Cleanup workspace always

	// Step 2: Write source code
	sourcePath := filepath.Join(workspace, "Main.java")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0777); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to write source file: %w", err)
	}
	// Give full permissions so Docker non-root user can read/write to this mounted dir
	_ = os.Chmod(workspace, 0777)
	_ = os.Chmod(sourcePath, 0666)

	// Step 3: Start Sandbox Container
	containerID, err := j.sandbox.StartContainer(ctx, workspace)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to start sandbox: %w", err)
	}
	defer j.sandbox.DestroyContainer(containerID)

	// Step 4: Compile
	compileOpts := sandbox.ExecOptions{
		Command:     "javac",
		Args:        []string{"-source", "1.8", "-target", "1.8", "Main.java"},
		StdoutLimit: j.config.MaxStdoutBytes,
		StderrLimit: j.config.MaxStderrBytes,
		Timeout:     j.config.CompileTimeout,
	}

	compileRes := j.sandbox.Exec(ctx, containerID, compileOpts)
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

	runOpts := sandbox.ExecOptions{
		Command:     "java",
		Args:        []string{"Main"},
		Stdin:       stdinReader,
		StdoutLimit: j.config.MaxStdoutBytes,
		StderrLimit: j.config.MaxStderrBytes,
		Timeout:     j.config.RunTimeout,
	}

	runRes := j.sandbox.Exec(ctx, containerID, runOpts)
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

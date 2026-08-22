package execution

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codera/code-executor/internal/domain"
)

type JavaExecutor struct{}

func NewJavaExecutor() *JavaExecutor {
	return &JavaExecutor{}
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
	javacCmd := exec.CommandContext(ctx, "javac", "-source", "1.8", "-target", "1.8", "Main.java")
	javacCmd.Dir = workspace

	var compileStderr bytes.Buffer
	javacCmd.Stderr = &compileStderr
	// Standard output usually empty for javac, but we can capture it if needed
	var compileStdout bytes.Buffer
	javacCmd.Stdout = &compileStdout

	err = javacCmd.Run()
	if err != nil {
		// Compilation failed
		exitCode := 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		return domain.ExecutionResult{
			Status:   domain.StatusCompilationError,
			Stdout:   compileStdout.String(),
			Stderr:   compileStderr.String(),
			ExitCode: exitCode,
		}, nil // Note: nil error because it's a user failure, not platform failure
	}

	// Step 5: Run
	javaCmd := exec.CommandContext(ctx, "java", "Main")
	javaCmd.Dir = workspace

	if req.Input != "" {
		javaCmd.Stdin = strings.NewReader(req.Input)
	}

	var runStdout bytes.Buffer
	var runStderr bytes.Buffer
	javaCmd.Stdout = &runStdout
	javaCmd.Stderr = &runStderr

	err = javaCmd.Run()
	if err != nil {
		exitCode := 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		return domain.ExecutionResult{
			Status:   domain.StatusRuntimeError,
			Stdout:   runStdout.String(),
			Stderr:   runStderr.String(),
			ExitCode: exitCode,
		}, nil
	}

	// Step 6: Classify result as SUCCESS
	return domain.ExecutionResult{
		Status:   domain.StatusSuccess,
		Stdout:   runStdout.String(),
		Stderr:   runStderr.String(),
		ExitCode: 0,
	}, nil
}

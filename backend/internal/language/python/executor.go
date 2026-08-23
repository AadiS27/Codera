package python

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox"
)

type Executor struct {
	profile sandbox.Profile
}

func NewExecutor(profile sandbox.Profile) *Executor {
	return &Executor{
		profile: profile,
	}
}

func (e *Executor) Language() domain.Language {
	return domain.LanguagePython
}

func (e *Executor) Profile() sandbox.Profile {
	return e.profile
}

func (e *Executor) Validate(req domain.ExecutionRequest) error {
	if strings.TrimSpace(req.SourceCode) == "" {
		return fmt.Errorf("source code cannot be empty")
	}
	return nil
}

func (e *Executor) Compile(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (*sandbox.ExecResult, error) {
	// Step 1: Write source code to workspace
	sourcePath := filepath.Join(workspaceDir, "main.py")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0777); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}
	_ = os.Chmod(workspaceDir, 0777)
	_ = os.Chmod(sourcePath, 0666)

	// Step 2: Validate syntax via py_compile
	compileOpts := sandbox.ExecOptions{
		Command:     "python",
		Args:        []string{"-m", "py_compile", "main.py"},
		StdoutLimit: e.profile.MaxOutputBytes,
		StderrLimit: e.profile.MaxOutputBytes,
		Timeout:     e.profile.Timeout, // Syntax check is fast
	}

	res := sb.Exec(ctx, containerID, compileOpts)
	return &res, nil
}

func (e *Executor) Execute(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (sandbox.ExecResult, error) {
	var stdinReader io.Reader
	if req.Input != "" {
		stdinReader = strings.NewReader(req.Input)
	}

	runOpts := sandbox.ExecOptions{
		Command:     "python",
		Args:        []string{"main.py"},
		Stdin:       stdinReader,
		StdoutLimit: e.profile.MaxOutputBytes,
		StderrLimit: e.profile.MaxOutputBytes,
		Timeout:     e.profile.Timeout,
	}

	res := sb.Exec(ctx, containerID, runOpts)
	return res, nil
}

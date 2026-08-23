package java

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
	return domain.LanguageJava
}

func (e *Executor) Profile() sandbox.Profile {
	return e.profile
}

func (e *Executor) Validate(req domain.ExecutionRequest) error {
	if strings.TrimSpace(req.SourceCode) == "" {
		return fmt.Errorf("source code cannot be empty")
	}
	if !strings.Contains(req.SourceCode, "public class Main") {
		return fmt.Errorf("java source must contain 'public class Main'")
	}
	return nil
}

func (e *Executor) Compile(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (*sandbox.ExecResult, error) {
	// Step 1: Write source code to workspace
	sourcePath := filepath.Join(workspaceDir, "Main.java")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCode), 0777); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}
	// Give full permissions so Docker non-root user can read/write to this mounted dir
	_ = os.Chmod(workspaceDir, 0777)
	_ = os.Chmod(sourcePath, 0666)

	// Step 2: Compile
	compileOpts := sandbox.ExecOptions{
		Command:     "javac",
		Args:        []string{"-source", "1.8", "-target", "1.8", "Main.java"},
		StdoutLimit: e.profile.MaxOutputBytes,
		StderrLimit: e.profile.MaxOutputBytes,
		// Compile usually has a higher timeout, let's use a multiple of run timeout for now
		// or wait, where do we get compile timeout? 
		// For simplicity, let's just use e.profile.Timeout * 2. In real scenarios it would be separate.
		Timeout:     e.profile.Timeout * 2, 
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
		Command:     "java",
		Args:        []string{"Main"},
		Stdin:       stdinReader,
		StdoutLimit: e.profile.MaxOutputBytes,
		StderrLimit: e.profile.MaxOutputBytes,
		Timeout:     e.profile.Timeout,
	}

	res := sb.Exec(ctx, containerID, runOpts)
	return res, nil
}

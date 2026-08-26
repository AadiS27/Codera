package sql

import (
	"context"
	"fmt"
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
	return domain.LanguageSql
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
	// SQL doesn't require compilation, we will stream the query directly to stdin during execution
	return nil, nil
}

func (e *Executor) Execute(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (sandbox.ExecResult, error) {
	fullScript := req.Input
	if fullScript != "" && !strings.HasSuffix(fullScript, "\n") {
		fullScript += "\n"
	}
	fullScript += req.SourceCode

	stdinReader := strings.NewReader(fullScript)

	runOpts := sandbox.ExecOptions{
		Command:     "sqlite3",
		Args:        []string{"-header", "-list", "-separator", "|", ":memory:"},
		Stdin:       stdinReader,
		StdoutLimit: e.profile.MaxOutputBytes,
		StderrLimit: e.profile.MaxOutputBytes,
		Timeout:     e.profile.Timeout,
	}

	res := sb.Exec(ctx, containerID, runOpts)
	return res, nil
}

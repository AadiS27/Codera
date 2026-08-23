package language

import (
	"context"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox"
)

// LanguageExecutor defines the contract for language-specific compilation and execution pipelines.
// This implements the Strategy Pattern, separating language logic from the worker queue logic.
type LanguageExecutor interface {
	// Language returns the strong language type this executor supports.
	Language() domain.Language

	// Profile returns the sandbox profile constraints for this language (image, memory limits, etc).
	Profile() sandbox.Profile

	// Validate checks if the request is structurally valid for this language (e.g. entrypoint, non-empty code).
	Validate(req domain.ExecutionRequest) error

	// Compile prepares the submitted source for execution. For interpreted languages, this might just validate syntax.
	// We return a pointer to ExecResult so we know if compilation actually ran and failed, or if it was bypassed.
	Compile(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (*sandbox.ExecResult, error)

	// Execute runs the prepared program inside the sandbox container.
	Execute(ctx context.Context, req domain.ExecutionRequest, workspaceDir string, sb sandbox.Runtime, containerID string) (sandbox.ExecResult, error)
}

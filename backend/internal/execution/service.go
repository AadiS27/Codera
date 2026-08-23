package execution

import (
	"context"
	"fmt"
	"os"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/sandbox"
	"github.com/codera/code-executor/internal/sandbox/classifier"
)

type Service struct {
	config   *config.Config
	registry language.Registry
	sandbox  sandbox.Runtime
}

func NewService(cfg *config.Config, registry language.Registry, sb sandbox.Runtime) *Service {
	return &Service{
		config:   cfg,
		registry: registry,
		sandbox:  sb,
	}
}

func (s *Service) Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	// 1. Get the language executor
	executor, err := s.registry.Get(req.Language)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to get language executor: %w", err)
	}

	// 2. Validate the request
	if err := executor.Validate(req); err != nil {
		return domain.ExecutionResult{
			Status: domain.StatusCompilationError,
			Stderr: "Validation Error: " + err.Error(),
		}, nil
	}

	// 3. Create an isolated workspace
	workspace, err := os.MkdirTemp("", "code-executor-"+string(req.Language)+"-*")
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workspace)

	// 4. Start sandbox container
	containerID, err := s.sandbox.StartContainer(ctx, workspace, executor.Profile())
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("failed to start sandbox: %w", err)
	}
	defer s.sandbox.DestroyContainer(containerID)

	// 5. Compile (or prepare/validate syntax)
	compileRes, err := executor.Compile(ctx, req, workspace, s.sandbox, containerID)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("internal compile error: %w", err)
	}

	if compileRes != nil {
		compileDomainRes := classifier.Classify(*compileRes)
		if compileDomainRes.Status != domain.StatusSuccess {
			// Map RUNTIME_ERROR to COMPILATION_ERROR since it failed during compile phase
			if compileDomainRes.Status == domain.StatusRuntimeError {
				compileDomainRes.Status = domain.StatusCompilationError
			} else if compileDomainRes.Status == domain.StatusTimeLimitExceeded {
				compileDomainRes.Status = domain.StatusCompilationTimeout
			}
			return compileDomainRes, nil
		}
	}

	// 6. Execute
	runRes, err := executor.Execute(ctx, req, workspace, s.sandbox, containerID)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("internal execution error: %w", err)
	}

	// 7. Classify result
	runDomainRes := classifier.Classify(runRes)

	return runDomainRes, nil
}

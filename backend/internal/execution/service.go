package execution

import (
	"context"
	"strings"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox"
)

type Executor interface {
	Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error)
}

type Service struct {
	config    *config.Config
	executors map[string]Executor
}

func NewService(cfg *config.Config, sb sandbox.Runtime) *Service {
	return &Service{
		config: cfg,
		executors: map[string]Executor{
			"java": NewJavaExecutor(cfg, sb),
		},
	}
}

func (s *Service) Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error) {
	lang := strings.ToLower(req.Language)
	if lang != "java" {
		return domain.ExecutionResult{}, ErrUnsupportedLanguage
	}

	if strings.TrimSpace(req.SourceCode) == "" {
		return domain.ExecutionResult{}, ErrEmptySourceCode
	}

	executor, ok := s.executors[lang]
	if !ok {
		return domain.ExecutionResult{}, ErrUnsupportedLanguage
	}

	return executor.Execute(ctx, req)
}

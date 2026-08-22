package execution

import (
	"context"
	"strings"

	"github.com/codera/code-executor/internal/domain"
)

type Executor interface {
	Execute(ctx context.Context, req domain.ExecutionRequest) (domain.ExecutionResult, error)
}

type Service struct {
	executors map[string]Executor
}

func NewService() *Service {
	return &Service{
		executors: map[string]Executor{
			"java": NewJavaExecutor(),
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

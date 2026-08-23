package language

import (
	"errors"
	"sync"

	"github.com/codera/code-executor/internal/domain"
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
)

// Registry defines the contract for a language registry.
type Registry interface {
	// Register adds a LanguageExecutor to the registry.
	Register(executor LanguageExecutor)
	// Get retrieves a LanguageExecutor by its Language.
	Get(lang domain.Language) (LanguageExecutor, error)
	// SupportedLanguages returns a list of all supported languages.
	SupportedLanguages() []domain.Language
}

type MapRegistry struct {
	mu        sync.RWMutex
	executors map[domain.Language]LanguageExecutor
}

func NewMapRegistry() *MapRegistry {
	return &MapRegistry{
		executors: make(map[domain.Language]LanguageExecutor),
	}
}

func (r *MapRegistry) Register(executor LanguageExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[executor.Language()] = executor
}

func (r *MapRegistry) Get(lang domain.Language) (LanguageExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, exists := r.executors[lang]
	if !exists {
		return nil, ErrUnsupportedLanguage
	}

	return executor, nil
}

func (r *MapRegistry) SupportedLanguages() []domain.Language {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var langs []domain.Language
	for lang := range r.executors {
		langs = append(langs, lang)
	}
	return langs
}

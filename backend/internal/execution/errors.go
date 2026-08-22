package execution

import "errors"

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
	ErrEmptySourceCode     = errors.New("source code cannot be empty")
	ErrOutputLimitExceeded = errors.New("output limit exceeded")
)

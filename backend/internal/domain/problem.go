package domain

import "time"

type ProblemStatus string

const (
	ProblemStatusDraft     ProblemStatus = "DRAFT"
	ProblemStatusPublished ProblemStatus = "PUBLISHED"
	ProblemStatusArchived  ProblemStatus = "ARCHIVED"
)

type ComparisonMode string

const (
	ComparisonModeExact      ComparisonMode = "EXACT"
	ComparisonModeWhitespace ComparisonMode = "NORMALIZED_WHITESPACE"
	ComparisonModeFloat      ComparisonMode = "FLOAT_EPSILON"
)

type Problem struct {
	ID               string
	Title            string
	Slug             string
	Description      string
	InputDescription string
	OutputDescription string
	Constraints      string
	
	TimeLimitMs      int
	MemoryLimitMB    int
	
	ComparisonMode   ComparisonMode
	FloatEpsilon     float64 // Used if ComparisonMode == FLOAT_EPSILON
	
	Status           ProblemStatus
	CreatedAt        time.Time
}

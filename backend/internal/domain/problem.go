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
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Slug              string         `json:"slug"`
	Description       string         `json:"description"`
	InputDescription  string         `json:"input_description"`
	OutputDescription string         `json:"output_description"`
	Constraints       string         `json:"constraints"`
	
	TimeLimitMs       int            `json:"time_limit_ms"`
	MemoryLimitMB     int            `json:"memory_limit_mb"`
	
	ComparisonMode    ComparisonMode `json:"comparison_mode"`
	FloatEpsilon      float64        `json:"float_epsilon"`
	
	Status            ProblemStatus  `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
}

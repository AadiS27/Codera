package domain

import "time"

type TestCaseVisibility string

const (
	VisibilityPublic TestCaseVisibility = "PUBLIC"
	VisibilityHidden TestCaseVisibility = "HIDDEN"
)

type TestCase struct {
	ID             string             `json:"id"`
	ProblemID      string             `json:"problem_id"`
	Input          string             `json:"input"`
	ExpectedOutput string             `json:"expected_output"`
	Visibility     TestCaseVisibility `json:"visibility"`
	SortOrder      int                `json:"sort_order"`
	CreatedAt      time.Time          `json:"created_at"`
}

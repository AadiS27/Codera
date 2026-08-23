package domain

import "time"

type TestCaseVisibility string

const (
	VisibilityPublic TestCaseVisibility = "PUBLIC"
	VisibilityHidden TestCaseVisibility = "HIDDEN"
)

type TestCase struct {
	ID             string
	ProblemID      string
	Input          string
	ExpectedOutput string
	Visibility     TestCaseVisibility
	SortOrder      int
	CreatedAt      time.Time
}

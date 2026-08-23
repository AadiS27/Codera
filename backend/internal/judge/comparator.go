package judge

import (
	"math"
	"strconv"
	"strings"

	"github.com/codera/code-executor/internal/domain"
)

// OutputComparator determines if the actual output matches the expected output.
type OutputComparator interface {
	Compare(expected, actual string) bool
}

// ComparatorRegistry stores and provides OutputComparators based on the comparison mode.
type ComparatorRegistry struct {
	comparators map[domain.ComparisonMode]OutputComparator
}

func NewComparatorRegistry() *ComparatorRegistry {
	return &ComparatorRegistry{
		comparators: make(map[domain.ComparisonMode]OutputComparator),
	}
}

func (r *ComparatorRegistry) Register(mode domain.ComparisonMode, c OutputComparator) {
	r.comparators[mode] = c
}

// Get returns the registered comparator. If not found or if the mode is unknown, it defaults to ExactComparator.
func (r *ComparatorRegistry) Get(mode domain.ComparisonMode) OutputComparator {
	c, ok := r.comparators[mode]
	if !ok {
		return &ExactComparator{} // Fallback
	}
	return c
}

// ExactComparator expects a byte-for-byte exact string match.
type ExactComparator struct{}

func (c *ExactComparator) Compare(expected, actual string) bool {
	return expected == actual
}

// WhitespaceComparator normalizes \r\n to \n, trims trailing whitespace per line,
// and ignores trailing empty lines at the end of the output.
type WhitespaceComparator struct{}

func (c *WhitespaceComparator) Compare(expected, actual string) bool {
	normExpected := c.normalize(expected)
	normActual := c.normalize(actual)
	return normExpected == normActual
}

func (c *WhitespaceComparator) normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	
	var normalizedLines []string
	for _, line := range lines {
		normalizedLines = append(normalizedLines, strings.TrimRight(line, " \t"))
	}
	
	// Remove trailing empty lines
	for len(normalizedLines) > 0 && normalizedLines[len(normalizedLines)-1] == "" {
		normalizedLines = normalizedLines[:len(normalizedLines)-1]
	}
	
	return strings.Join(normalizedLines, "\n")
}

// FloatComparator compares whitespace-separated numeric tokens with a given epsilon.
type FloatComparator struct {
	Epsilon float64
}

func (c *FloatComparator) Compare(expected, actual string) bool {
	expectedTokens := strings.Fields(expected)
	actualTokens := strings.Fields(actual)

	if len(expectedTokens) != len(actualTokens) {
		return false
	}

	for i := range expectedTokens {
		expVal, err1 := strconv.ParseFloat(expectedTokens[i], 64)
		actVal, err2 := strconv.ParseFloat(actualTokens[i], 64)

		if err1 != nil || err2 != nil {
			// If they aren't floats, fallback to exact string match for this token
			if expectedTokens[i] != actualTokens[i] {
				return false
			}
			continue
		}

		if math.Abs(expVal-actVal) > c.Epsilon {
			return false
		}
	}

	return true
}

// FloatComparatorFactory creates a new FloatComparator for a specific problem epsilon.
type FloatComparatorFactory struct{}

func (f *FloatComparatorFactory) Create(epsilon float64) OutputComparator {
	return &FloatComparator{Epsilon: epsilon}
}

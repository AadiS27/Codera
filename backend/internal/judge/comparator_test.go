package judge

import (
	"testing"
)

func TestExactComparator(t *testing.T) {
	comp := &ExactComparator{}

	if !comp.Compare("hello\n", "hello\n") {
		t.Error("Expected exact match to pass")
	}

	if comp.Compare("hello\n", "hello\r\n") {
		t.Error("Expected exact match with different line endings to fail")
	}
}

func TestWhitespaceComparator(t *testing.T) {
	comp := &WhitespaceComparator{}

	if !comp.Compare("hello\nworld", "hello\r\nworld") {
		t.Error("Expected whitespace comparator to ignore CRLF vs LF")
	}

	if !comp.Compare("hello  \nworld", "hello\nworld") {
		t.Error("Expected whitespace comparator to ignore trailing spaces")
	}

	if !comp.Compare("hello\nworld\n", "hello\nworld") {
		t.Error("Expected whitespace comparator to ignore trailing empty lines")
	}

	if !comp.Compare("hello\nworld\n\n\n", "hello\nworld\n") {
		t.Error("Expected whitespace comparator to ignore multiple trailing empty lines")
	}

	if comp.Compare("hello \n world", "hello\nworld") {
		t.Error("Expected whitespace comparator NOT to ignore leading spaces")
	}
}

func TestFloatComparator(t *testing.T) {
	comp := &FloatComparator{Epsilon: 1e-4}

	if !comp.Compare("1.0001", "1.00015") {
		t.Error("Expected float comparator to pass within epsilon")
	}

	if comp.Compare("1.0001", "1.0003") {
		t.Error("Expected float comparator to fail outside epsilon")
	}

	if !comp.Compare("ans: 1.5", "ans: 1.50001") {
		t.Error("Expected mixed string and float comparison to pass")
	}

	if comp.Compare("wrong: 1.5", "ans: 1.5") {
		t.Error("Expected mismatched string tokens to fail")
	}
}

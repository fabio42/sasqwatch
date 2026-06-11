package ui

import (
	"strings"
	"testing"

	"github.com/fabio42/sasqwatch/ui/theme"
)

func newStatusModel(width int) Model {
	m := newTestModel(5)
	m.width = width
	m.cfg.Theme = theme.DefaultTheme()
	return m
}

func TestTruncStatus_NoTruncation(t *testing.T) {
	m := newStatusModel(80)
	input := "hello"
	// Reserve 5 columns for the right-hand side; with width=80 that leaves 75 — plenty.
	result := m.truncStatus(input, 5)
	if result != input {
		t.Fatalf("expected no truncation, got %q", result)
	}
}

func TestTruncStatus_Truncates_AndAddsEllipsis(t *testing.T) {
	m := newStatusModel(20)
	// right-side reservation = 18; left gets 2 columns, so a 10-char string must be cut.
	input := "0123456789"
	result := m.truncStatus(input, 18)
	if !strings.HasSuffix(result, "…") {
		t.Fatalf("expected ellipsis at end of truncated string, got %q", result)
	}
	if len([]rune(result)) >= len([]rune(input)) {
		t.Fatalf("expected result shorter than input, got %q (len=%d)", result, len([]rune(result)))
	}
}

func TestTruncStatus_SmallWidth_SingleRune(t *testing.T) {
	m := newStatusModel(5)
	// Extreme case: almost no room.
	input := "abcdefghij"
	result := m.truncStatus(input, 4)
	// Should not panic and should return something.
	if result == "" {
		t.Fatal("expected non-empty result even for very small width")
	}
}

func TestTruncStatus_Empty_Input(t *testing.T) {
	m := newStatusModel(80)
	result := m.truncStatus("", 5)
	if result != "" {
		t.Fatalf("expected empty result for empty input, got %q", result)
	}
}

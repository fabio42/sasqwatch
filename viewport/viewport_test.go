package viewport

import "testing"

func newTestViewport(width, height int, content string) Model {
	m := New(width, height)
	m.SetContent(content)
	return m
}

// --- vertical scrolling ---

func TestLineDown_MovesOffset(t *testing.T) {
	m := newTestViewport(20, 3, "line1\nline2\nline3\nline4\nline5")
	if m.YOffset != 0 {
		t.Fatalf("expected initial YOffset=0, got %d", m.YOffset)
	}
	m.LineDown(1)
	if m.YOffset != 1 {
		t.Fatalf("expected YOffset=1 after LineDown(1), got %d", m.YOffset)
	}
}

func TestLineDown_ClampsAtBottom(t *testing.T) {
	m := newTestViewport(20, 3, "a\nb\nc\nd")
	m.LineDown(100)
	if m.YOffset != m.maxYOffset() {
		t.Fatalf("expected YOffset clamped to %d, got %d", m.maxYOffset(), m.YOffset)
	}
}

func TestLineUp_MovesOffset(t *testing.T) {
	m := newTestViewport(20, 3, "a\nb\nc\nd\ne")
	m.LineDown(2)
	m.LineUp(1)
	if m.YOffset != 1 {
		t.Fatalf("expected YOffset=1 after LineDown(2)+LineUp(1), got %d", m.YOffset)
	}
}

func TestLineUp_ClampsAtTop(t *testing.T) {
	m := newTestViewport(20, 3, "a\nb\nc")
	m.LineDown(1)
	m.LineUp(100)
	if m.YOffset != 0 {
		t.Fatalf("expected YOffset=0 after LineUp past top, got %d", m.YOffset)
	}
}

func TestAtTop_AtBottom(t *testing.T) {
	m := newTestViewport(20, 2, "a\nb\nc")
	if !m.AtTop() {
		t.Fatal("expected AtTop() on fresh viewport")
	}
	m.GotoBottom()
	if !m.AtBottom() {
		t.Fatal("expected AtBottom() after GotoBottom()")
	}
}

// --- horizontal scrolling ---

func TestMoveRight_IncreasesIndent(t *testing.T) {
	m := newTestViewport(20, 5, "hello world")
	m.MoveRight()
	if m.indent != defaultHorizontalStep {
		t.Fatalf("expected indent=%d after MoveRight, got %d", defaultHorizontalStep, m.indent)
	}
}

func TestMoveLeft_DecreasesIndent(t *testing.T) {
	m := newTestViewport(20, 5, "hello world")
	m.MoveRight()
	m.MoveRight()
	m.MoveLeft()
	if m.indent != defaultHorizontalStep {
		t.Fatalf("expected indent=%d after 2xRight+1xLeft, got %d", defaultHorizontalStep, m.indent)
	}
}

func TestMoveLeft_ClampsAtZero(t *testing.T) {
	m := newTestViewport(20, 5, "hello world")
	// Should not go negative.
	m.MoveLeft()
	m.MoveLeft()
	if m.indent != 0 {
		t.Fatalf("expected indent clamped to 0, got %d", m.indent)
	}
}

func TestResetIndent(t *testing.T) {
	m := newTestViewport(20, 5, "hello world")
	m.MoveRight()
	m.MoveRight()
	m.ResetIndent()
	if m.indent != 0 {
		t.Fatalf("expected indent=0 after ResetIndent, got %d", m.indent)
	}
}

func TestSetHorizontalStep_Negative_ClampedToZero(t *testing.T) {
	m := newTestViewport(20, 5, "content")
	m.SetHorizontalStep(-5)
	if m.horizontalStep != 0 {
		t.Fatalf("expected horizontalStep=0 for negative input, got %d", m.horizontalStep)
	}
}

// --- scroll percent ---

func TestScrollPercent_FullyVisible(t *testing.T) {
	m := newTestViewport(20, 10, "a\nb")
	if m.ScrollPercent() != 1.0 {
		t.Fatalf("expected ScrollPercent=1.0 when content fits, got %f", m.ScrollPercent())
	}
}

func TestScrollPercent_Clamped(t *testing.T) {
	m := newTestViewport(20, 2, "a\nb\nc\nd\ne")
	p := m.ScrollPercent()
	if p < 0 || p > 1 {
		t.Fatalf("ScrollPercent must be in [0,1], got %f", p)
	}
}

// --- clamp helper ---

func TestClamp(t *testing.T) {
	cases := []struct{ v, low, high, want int }{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{11, 0, 10, 10},
		{5, 10, 0, 5}, // inverted low/high should swap
	}
	for _, c := range cases {
		if got := clamp(c.v, c.low, c.high); got != c.want {
			t.Errorf("clamp(%d,%d,%d)=%d, want %d", c.v, c.low, c.high, got, c.want)
		}
	}
}

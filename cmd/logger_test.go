package cmd

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
)

func TestLevelWriter_AboveThreshold_Writes(t *testing.T) {
	var buf bytes.Buffer
	lw := &LevelWriter{Writer: &buf, Level: zerolog.WarnLevel}

	msg := []byte("warn message")
	n, err := lw.WriteLevel(zerolog.WarnLevel, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(msg) {
		t.Fatalf("expected %d bytes written, got %d", len(msg), n)
	}
	if buf.String() != string(msg) {
		t.Fatalf("expected %q in buffer, got %q", msg, buf.String())
	}
}

func TestLevelWriter_AtThreshold_Writes(t *testing.T) {
	var buf bytes.Buffer
	lw := &LevelWriter{Writer: &buf, Level: zerolog.ErrorLevel}

	msg := []byte("error message")
	lw.WriteLevel(zerolog.ErrorLevel, msg)
	if buf.Len() == 0 {
		t.Fatal("expected message to be written at exactly the threshold level")
	}
}

func TestLevelWriter_BelowThreshold_Discards(t *testing.T) {
	var buf bytes.Buffer
	lw := &LevelWriter{Writer: &buf, Level: zerolog.WarnLevel}

	msg := []byte("debug message")
	n, err := lw.WriteLevel(zerolog.DebugLevel, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(msg) {
		t.Fatalf("WriteLevel should return len(p) even when discarding, got %d", n)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected nothing written below threshold, but buffer contains %q", buf.String())
	}
}

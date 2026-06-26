package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOutcomeLine_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"mission_id":"m-1","status":"completed","timestamp":"2026-06-02T12:00:00Z"}`,
		`{"mission_id":"m-2","status":"analysis_delivered","timestamp":"2026-06-02T13:00:00Z","gates":[{"type":"approval_gate","approved_at":"2026-06-02T13:01:00Z","response":"sim"}]}`,
	}
	for _, line := range cases {
		if err := ValidateOutcomeLine(line); err != nil {
			t.Fatalf("unexpected error for valid line %q: %v", line, err)
		}
	}
}

func TestValidateOutcomeLine_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		line    string
		missing string
	}{
		{`{"status":"completed","timestamp":"2026-06-02T12:00:00Z"}`, "mission_id"},
		{`{"mission_id":"m-1","timestamp":"2026-06-02T12:00:00Z"}`, "status"},
		{`{"mission_id":"m-1","status":"completed"}`, "timestamp"},
		{`{}`, "mission_id"},
	}
	for _, tc := range cases {
		err := ValidateOutcomeLine(tc.line)
		if err == nil {
			t.Fatalf("expected error for missing %q, got nil", tc.missing)
		}
		if !strings.Contains(err.Error(), tc.missing) {
			t.Fatalf("error should mention %q: %v", tc.missing, err)
		}
	}
}

func TestValidateOutcomeLine_InvalidJSON(t *testing.T) {
	t.Parallel()
	if err := ValidateOutcomeLine("not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAppendOutcomeLine_WritesValidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.tmp")
	line := `{"mission_id":"m-3","status":"completed","timestamp":"2026-06-02T14:00:00Z"}`

	if err := AppendOutcomeLine(path, line); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if !strings.Contains(string(data), line) {
		t.Fatalf("line not found in file: %s", data)
	}
}

func TestAppendOutcomeLine_RejectsInvalidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.tmp")

	err := AppendOutcomeLine(path, `{"mission_id":"","status":"","timestamp":""}`)
	if err == nil {
		t.Fatal("expected error for invalid line")
	}

	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("file should not be created on validation failure")
	}
}

func TestAppendOutcomeLine_OpenError(t *testing.T) {
	t.Parallel()
	// Path in a nonexistent directory — os.OpenFile must fail.
	path := filepath.Join(t.TempDir(), "nonexistent-subdir", "outcomes.tmp")
	line := `{"mission_id":"m-err","status":"completed","timestamp":"2026-06-02T14:00:00Z"}`
	err := AppendOutcomeLine(path, line)
	if err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
}

func TestAppendOutcomeLine_Idempotent_MultipleLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.tmp")

	for i := range 3 {
		_ = i
		line := `{"mission_id":"m-loop","status":"completed","timestamp":"2026-06-02T15:00:00Z"}`
		if err := AppendOutcomeLine(path, line); err != nil {
			t.Fatalf("append error: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	lineCount := strings.Count(string(data), "\n")
	if lineCount != 3 {
		t.Fatalf("expected 3 lines, got %d", lineCount)
	}
}

func TestAppendOutcomeLineSafe_DoesNotPanicOnBadPath(t *testing.T) {
	t.Parallel()
	// Must not panic or return error — learning failures are non-blocking.
	validLine := `{"mission_id":"m-safe","status":"completed","timestamp":"2026-06-23T00:00:00Z"}`
	AppendOutcomeLineSafe("/nonexistent/path/that/cannot/exist/outcomes.jsonl", validLine)
}

func TestAppendOutcomeLineSafe_WritesOnValidPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.jsonl")
	line := `{"mission_id":"m-safe-2","status":"completed","timestamp":"2026-06-23T00:00:00Z"}`
	AppendOutcomeLineSafe(path, line)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if !strings.Contains(string(data), "m-safe-2") {
		t.Errorf("expected written content, got: %s", string(data))
	}
}

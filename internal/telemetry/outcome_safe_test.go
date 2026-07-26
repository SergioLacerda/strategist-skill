package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestAppendOutcomeLineSafe_LogsDebugOnDuplicate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.jsonl")
	line := `{"mission_id":"m-safe-dup","status":"completed","timestamp":"2026-06-23T00:00:00Z"}`
	AppendOutcomeLineSafe(path, line)
	// Second call with the same mission_id hits the "duplicate, not appended" branch.
	AppendOutcomeLineSafe(path, line)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Count(string(data), "m-safe-dup") != 1 {
		t.Fatalf("expected exactly one entry, got: %s", data)
	}
}

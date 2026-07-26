package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlushOutcomeBuffer_AbsentBufferIsNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bufferPath := filepath.Join(dir, "outcomes.tmp")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	flushed, err := FlushOutcomeBuffer(bufferPath, outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flushed != 0 {
		t.Fatalf("expected 0 flushed, got %d", flushed)
	}
}

func TestFlushOutcomeBuffer_MovesBufferedLinesAndClearsBuffer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bufferPath := filepath.Join(dir, "outcomes.tmp")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	buffered := `{"mission_id":"m-flush-1","status":"completed","timestamp":"2026-07-22T10:00:00Z"}` + "\n" +
		`{"mission_id":"m-flush-2","status":"completed","timestamp":"2026-07-22T10:05:00Z"}` + "\n"
	writeOutcomeTestFile(t, bufferPath, buffered)

	flushed, err := FlushOutcomeBuffer(bufferPath, outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flushed != 2 {
		t.Fatalf("expected 2 flushed, got %d", flushed)
	}

	assertOutcomeTestFileEmpty(t, bufferPath)
	assertOutcomeTestFileContains(t, outcomesPath, "m-flush-1", "m-flush-2")
}

func TestFlushOutcomeBuffer_SkipsEntryAlreadyInOutcomes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bufferPath := filepath.Join(dir, "outcomes.tmp")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	// Simulates a flush that was interrupted after the append but before the
	// buffer truncate (ADR-0004): the entry is already in outcomes.jsonl but
	// the buffer still holds it too.
	existing := `{"mission_id":"m-dup","status":"completed","timestamp":"2026-07-22T09:00:00Z"}` + "\n"
	if err := os.WriteFile(outcomesPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("setup outcomes: %v", err)
	}
	if err := os.WriteFile(bufferPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("setup buffer: %v", err)
	}

	flushed, err := FlushOutcomeBuffer(bufferPath, outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flushed != 0 {
		t.Fatalf("expected 0 flushed (duplicate skipped), got %d", flushed)
	}

	outData, err := os.ReadFile(outcomesPath)
	if err != nil {
		t.Fatalf("outcomes read error: %v", err)
	}
	lineCount := strings.Count(string(outData), "\n")
	if lineCount != 1 {
		t.Fatalf("expected outcomes.jsonl to still have 1 line, got %d: %s", lineCount, outData)
	}
}

func TestFlushOutcomeBuffer_PreservesJewelIDsAndSkipsDuplicateMissionID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bufferPath := filepath.Join(dir, "outcomes.tmp")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	existing := `{"mission_id":"m-jewel-dup","status":"completed","timestamp":"2026-07-25T09:00:00Z","jewel_ids":["old-jewel"]}` + "\n"
	buffered := `{"mission_id":"m-jewel-dup","status":"completed","timestamp":"2026-07-25T09:05:00Z","jewel_ids":["new-jewel"]}` + "\n" +
		`{"mission_id":"m-jewel-new","status":"completed","timestamp":"2026-07-25T09:10:00Z","jewel_ids":["jewel-a","jewel-b"]}` + "\n"
	writeOutcomeTestFile(t, outcomesPath, existing)
	writeOutcomeTestFile(t, bufferPath, buffered)

	flushed, err := FlushOutcomeBuffer(bufferPath, outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flushed != 1 {
		t.Fatalf("expected 1 flushed, got %d", flushed)
	}

	entries := readOutcomeTestEntries(t, outcomesPath)
	if len(entries) != 2 {
		t.Fatalf("expected 2 outcome entries, got %d", len(entries))
	}
	assertOutcomeTestEntryJewelIDs(t, entries, "m-jewel-dup", "old-jewel")
	assertOutcomeTestEntryJewelIDs(t, entries, "m-jewel-new", "jewel-a", "jewel-b")
}

func TestReadOutcomeBuffer_RealReadErrorNotMasked(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory path makes os.ReadFile fail with a real (non-ErrNotExist) error.
	_, err := readOutcomeBuffer(dir)
	if err == nil {
		t.Fatal("expected error reading a directory as a file, got nil")
	}
}

func TestFlushOutcomeBuffer_PropagatesReadBufferError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")
	// bufferPath is a directory, not a file — readOutcomeBuffer returns a real error.
	_, err := FlushOutcomeBuffer(dir, outcomesPath)
	if err == nil {
		t.Fatal("expected error when buffer path is a directory, got nil")
	}
}

func TestFlushOutcomeBuffer_SkipsBlankLineAndInvalidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bufferPath := filepath.Join(dir, "outcomes.tmp")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	buffered := "\n" + // blank line, must be skipped without error
		`{"mission_id":"","status":"","timestamp":""}` + "\n" + // fails validation
		`{"mission_id":"m-good","status":"completed","timestamp":"2026-07-25T00:00:00Z"}` + "\n"
	writeOutcomeTestFile(t, bufferPath, buffered)

	flushed, err := FlushOutcomeBuffer(bufferPath, outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flushed != 1 {
		t.Fatalf("expected 1 flushed (blank and invalid lines skipped), got %d", flushed)
	}
	assertOutcomeTestFileEmpty(t, bufferPath)
	assertOutcomeTestFileContains(t, outcomesPath, "m-good")
}

func TestTruncateOutcomeBuffer_ErrorOnMissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := truncateOutcomeBuffer(filepath.Join(dir, "does-not-exist"))
	if err == nil {
		t.Fatal("expected error truncating a nonexistent file, got nil")
	}
}

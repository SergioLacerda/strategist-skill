package telemetry

import (
	"errors"
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

	appended, err := AppendOutcomeLine(path, line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !appended {
		t.Fatal("expected line to be appended")
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

	_, err := AppendOutcomeLine(path, `{"mission_id":"","status":"","timestamp":""}`)
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
	_, err := AppendOutcomeLine(path, line)
	if err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
}

func TestAppendOutcomeLine_SkipsDuplicateMissionID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.tmp")
	line := `{"mission_id":"m-loop","status":"completed","timestamp":"2026-06-02T15:00:00Z"}`

	for i := range 3 {
		appended := appendOutcomeTestLine(t, path, line)
		wantAppended := i == 0
		if appended != wantAppended {
			t.Fatalf("append %d: got appended=%v, want %v", i, appended, wantAppended)
		}
	}

	assertOutcomeTestLineCount(t, path, 1)
}

func TestAppendOutcomeLine_SameStatusDifferentMissionIDBothAppend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.jsonl")

	first := `{"mission_id":"m-a","status":"completed","timestamp":"2026-06-02T15:00:00Z"}`
	second := `{"mission_id":"m-b","status":"completed","timestamp":"2026-06-02T15:05:00Z"}`

	for _, line := range []string{first, second} {
		if !appendOutcomeTestLine(t, path, line) {
			t.Fatalf("expected distinct mission_id %q to be appended", line)
		}
	}

	assertOutcomeTestLineCount(t, path, 2)
}

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

func TestCloseOutcomeFile_ReturnsErrorOnDoubleClose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "double-close"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	var outerErr error
	closeOutcomeFile(f, &outerErr)
	if outerErr == nil {
		t.Fatal("expected error on double close, got nil")
	}
}

func TestCloseOutcomeFile_DoesNotOverwriteExistingError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "existing-err"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	sentinel := errors.New("prior error")
	outerErr := sentinel
	closeOutcomeFile(f, &outerErr)
	if !errors.Is(outerErr, sentinel) {
		t.Fatalf("expected prior error preserved, got: %v", outerErr)
	}
}

func TestUnlockOutcomeFile_ReturnsErrorOnClosedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "unlock-closed"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	var outerErr error
	unlockOutcomeFile(f, &outerErr)
	if outerErr == nil {
		t.Fatal("expected error unlocking a closed file, got nil")
	}
}

func TestAppendOutcomeLineLocked_LockFailureOnClosedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "lock-closed"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	line := `{"mission_id":"m-lock","status":"completed","timestamp":"2026-07-25T00:00:00Z"}`
	_, err = appendOutcomeLineLocked(f, "m-lock", line)
	if err == nil {
		t.Fatal("expected lock error on closed file, got nil")
	}
}

func TestOutcomeLineHasMissionID_EmptyLine(t *testing.T) {
	t.Parallel()
	if outcomeLineHasMissionID(nil, "m-1") {
		t.Fatal("expected false for empty line")
	}
}

func TestOutcomeLineHasMissionID_MalformedJSONTolerated(t *testing.T) {
	t.Parallel()
	if outcomeLineHasMissionID([]byte("not json"), "m-1") {
		t.Fatal("expected false for malformed JSON, not an error")
	}
}

func TestOutcomeScannerErr_ReportsTooLongLine(t *testing.T) {
	t.Parallel()
	// A line exceeding the scanner's 1MB max triggers bufio.ErrTooLong.
	huge := strings.Repeat("x", 2*1024*1024)
	scanner := newOutcomeScanner(strings.NewReader(huge))
	for scanner.Scan() {
		// drain
	}
	err := outcomeScannerErr(scanner, "scan test")
	if err == nil {
		t.Fatal("expected scanner error for oversized line, got nil")
	}
	if !strings.Contains(err.Error(), "scan test") {
		t.Fatalf("expected error to include context, got: %v", err)
	}
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

func writeOutcomeTestFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write test file %s: %v", path, err)
	}
}

func appendOutcomeTestLine(t *testing.T, path, line string) bool {
	t.Helper()
	appended, err := AppendOutcomeLine(path, line)
	if err != nil {
		t.Fatalf("append outcome line: %v", err)
	}
	return appended
}

func assertOutcomeTestLineCount(t *testing.T, path string, want int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test file %s: %v", path, err)
	}
	if got := strings.Count(string(data), "\n"); got != want {
		t.Fatalf("expected %d lines in %s, got %d", want, path, got)
	}
}

func assertOutcomeTestFileEmpty(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test file %s: %v", path, err)
	}
	if len(data) != 0 {
		t.Fatalf("expected %s to be cleared, got %d bytes", path, len(data))
	}
}

func assertOutcomeTestFileContains(t *testing.T, path string, values ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test file %s: %v", path, err)
	}
	for _, value := range values {
		if !strings.Contains(string(data), value) {
			t.Fatalf("expected %s in %s, got: %s", value, path, data)
		}
	}
}

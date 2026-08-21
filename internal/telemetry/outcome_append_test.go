package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

func TestAppendOutcomeLine_PreservesJewelIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "outcomes.tmp")
	line := `{"mission_id":"m-jewels","status":"completed","timestamp":"2026-07-25T00:00:00Z","jewel_ids":["jewel-a","jewel-b"]}`

	appended, err := AppendOutcomeLine(path, line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !appended {
		t.Fatal("expected line to be appended")
	}

	entries := readOutcomeTestEntries(t, path)
	if got, want := strings.Join(entries[0].JewelIDs, ","), "jewel-a,jewel-b"; got != want {
		t.Fatalf("unexpected jewel_ids: got %q, want %q", got, want)
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

func TestUnlockOutcomeFile_ReturnsErrorOnClosedFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("unlocking a closed file can be a no-op on Windows")
	}
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

// TestAppendOutcomeLineLocked_SeekFailurePropagatesFromMissionIDExists uses the
// read end of an os.Pipe() as a file: Flock succeeds on a pipe fd (unlike a
// closed file, which fails locking before ever reaching missionIDExists), but
// Seek does not — pipes are not seekable. This isolates and covers both
// missionIDExists's own Seek-error branch and its propagation into
// appendOutcomeLineLocked, neither of which the closed-file lock-failure test
// above can reach (that one returns earlier, at lockFile).
func TestAppendOutcomeLineLocked_SeekFailurePropagatesFromMissionIDExists(t *testing.T) {
	t.Parallel()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer w.Close()
	defer r.Close()

	line := `{"mission_id":"m-seek","status":"completed","timestamp":"2026-07-25T00:00:00Z"}`
	_, err = appendOutcomeLineLocked(r, "m-seek", line)
	if err == nil {
		t.Fatal("expected error from non-seekable file, got nil")
	}
	if !strings.Contains(err.Error(), "scan outcomes file") {
		t.Fatalf("expected error to be wrapped by missionIDExists's scan-outcomes-file context, got: %v", err)
	}
}

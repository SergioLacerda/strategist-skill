package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadHandoffChallenges_MissingFileIsEmptyNotError(t *testing.T) {
	t.Parallel()
	records, err := ReadHandoffChallenges(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records, got %v", records)
	}
}

func TestReadHandoffChallenges_ParsesValidLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "handoff-challenges.jsonl")
	content := `{"mission_id":"m-1","transition":"archivist_to_sniper","attempt":1,"timestamp":"2026-08-03T00:00:00Z","status":"passed","passed":true,"critical_failures":0}
{"mission_id":"m-1","transition":"archivist_to_sniper","attempt":2,"timestamp":"2026-08-03T00:05:00Z","status":"failed","passed":false,"missing_refs":["X-001"],"critical_failures":1}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, err := ReadHandoffChallenges(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].MissionID != "m-1" || records[0].Attempt != 1 || !records[0].Passed {
		t.Fatalf("unexpected first record: %+v", records[0])
	}
	if records[1].Attempt != 2 || records[1].Passed {
		t.Fatalf("unexpected second record: %+v", records[1])
	}
}

func TestReadHandoffChallenges_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "handoff-challenges.jsonl")
	content := "not json\n" + `{"mission_id":"m-1","attempt":1,"passed":true}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, err := ReadHandoffChallenges(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record (malformed line skipped), got %d", len(records))
	}
}

func TestReadHandoffChallenges_SkipsBlankLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "handoff-challenges.jsonl")
	content := `{"mission_id":"m-1","attempt":1,"passed":true}` + "\n\n" +
		`{"mission_id":"m-2","attempt":1,"passed":false}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, err := ReadHandoffChallenges(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records (blank line skipped), got %d", len(records))
	}
}

func TestHandoffChallengeHistoryPath(t *testing.T) {
	t.Parallel()
	got := HandoffChallengeHistoryPath("/tmp/strategist-root")
	want := filepath.Join("/tmp/strategist-root", "memory", "handoff-challenges.jsonl")
	if got != want {
		t.Fatalf("unexpected path: got %q, want %q", got, want)
	}
}

func TestAppendHandoffChallenge_WritesAndReadsBack(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "memory", "handoff-challenges.jsonl")
	rec := ChallengeRecord{
		MissionID:  "m-1",
		Transition: "archivist_to_sniper",
		Attempt:    1,
		Timestamp:  "2026-08-05T00:00:00Z",
		Status:     "passed",
		Passed:     true,
	}

	if err := AppendHandoffChallenge(path, rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, err := ReadHandoffChallenges(path)
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].MissionID != "m-1" || !records[0].Passed {
		t.Fatalf("unexpected record: %+v", records[0])
	}
}

func TestAppendHandoffChallenge_MkdirAllFailsWhenParentIsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	// blocker is a file, so MkdirAll(blocker/memory, ...) must fail.
	path := filepath.Join(blocker, "memory", "handoff-challenges.jsonl")

	err := AppendHandoffChallenge(path, ChallengeRecord{MissionID: "m-1"})
	if err == nil {
		t.Fatal("expected error when parent dir cannot be created, got nil")
	}
}

func TestAppendHandoffChallenge_OpenFailsWhenPathIsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// path itself is an existing directory — os.OpenFile must fail.
	if err := os.Mkdir(filepath.Join(dir, "history-dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "history-dir")

	err := AppendHandoffChallenge(path, ChallengeRecord{MissionID: "m-1"})
	if err == nil {
		t.Fatal("expected error when path is a directory, got nil")
	}
}

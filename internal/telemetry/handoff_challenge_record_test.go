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

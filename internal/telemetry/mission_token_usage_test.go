package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissionTokenUsageHistoryPath(t *testing.T) {
	t.Parallel()
	got := MissionTokenUsageHistoryPath("/root")
	want := filepath.Join("/root", "memory", "mission-token-usage.jsonl")
	if got != want {
		t.Fatalf("MissionTokenUsageHistoryPath = %q, want %q", got, want)
	}
}

func TestReadMissionTokenUsage_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.jsonl")

	lines := []string{
		"not json at all",
		`{"mission_id":"m-1","tokens_in":10,"tokens_out":5,"source":"agent_report","reported_at":"2026-09-05T00:00:00Z"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := ReadMissionTokenUsage(path)
	if err != nil {
		t.Fatalf("ReadMissionTokenUsage: %v", err)
	}
	if len(got) != 1 || got[0].MissionID != "m-1" {
		t.Fatalf("expected only the well-formed record, got %+v", got)
	}
}

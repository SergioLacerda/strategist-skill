package telemetry

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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

func readOutcomeTestEntries(t *testing.T, path string) []OutcomeEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test file %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	entries := make([]OutcomeEntry, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry OutcomeEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("unmarshal outcome line %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func assertOutcomeTestEntryJewelIDs(t *testing.T, entries []OutcomeEntry, missionID string, want ...string) {
	t.Helper()
	for _, entry := range entries {
		if entry.MissionID != missionID {
			continue
		}
		got := strings.Join(entry.JewelIDs, ",")
		if got != strings.Join(want, ",") {
			t.Fatalf("mission %s jewel_ids: got %q, want %q", missionID, got, strings.Join(want, ","))
		}
		return
	}
	t.Fatalf("mission %s not found in entries: %#v", missionID, entries)
}

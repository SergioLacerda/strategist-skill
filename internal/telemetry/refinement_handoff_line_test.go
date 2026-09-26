package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }
func str(v string) *string   { return &v }
func intp(v int) *int        { return &v }
func handoffLinePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "memory", "handoff-metrics.jsonl")
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test-owned temp path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

// Unmeasured fields stay null, as the contract requires: they are expected to
// be null during rollout, never guessed.
func TestAppendRefinementHandoffLineWritesNullsForUnmeasuredFields(t *testing.T) {
	path := handoffLinePath(t)
	appended, err := AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: "m-1", RefinementReopens: intp(1), Model: str("m"), Effort: str("low"), LevelSource: str("host")})
	if err != nil || !appended {
		t.Fatalf("appended=%v err=%v", appended, err)
	}
	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("lines = %v", lines)
	}
	want := `{"mission_id":"m-1","discovery_tokens":null,"brief_tokens":null,"brief_compression_ratio":null,"refinement_reopens":1,"evidence_coverage_ratio":null,"model":"m","effort":"low","level_source":"host"}`
	if lines[0] != want {
		t.Fatalf("line =\n%s\nwant\n%s", lines[0], want)
	}
}

// An unmeasured reopen count is null, never a silent 0: 0 means the Archivist
// measured and found no reopen. A line without a revision carries no revision key.
func TestAppendRefinementHandoffLineKeepsReopensNullUntilMeasured(t *testing.T) {
	path := handoffLinePath(t)
	if _, err := AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: "unmeasured"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: "measured-zero", RefinementReopens: intp(0)}); err != nil {
		t.Fatal(err)
	}
	lines := readLines(t, path)
	if !strings.Contains(lines[0], `"refinement_reopens":null`) || strings.Contains(lines[0], `"revision"`) {
		t.Fatalf("unmeasured line = %s", lines[0])
	}
	if !strings.Contains(lines[1], `"refinement_reopens":0`) {
		t.Fatalf("measured zero line = %s", lines[1])
	}
}

// One line per revision: the base line and each revision are separate records of
// the same mission, and each is idempotent on its own.
func TestAppendRefinementHandoffLineRecordsOneLinePerRevision(t *testing.T) {
	path := handoffLinePath(t)
	for _, line := range []RefinementHandoffLine{
		{MissionID: "m-1"},
		{MissionID: "m-1", Revision: intp(1), RefinementReopens: intp(2)},
		{MissionID: "m-1", Revision: intp(1), RefinementReopens: intp(9)},
		{MissionID: "m-1", Revision: intp(2)},
		{MissionID: "m-1"},
	} {
		if _, err := AppendRefinementHandoffLine(path, line); err != nil {
			t.Fatal(err)
		}
	}
	lines := readLines(t, path)
	if len(lines) != 3 {
		t.Fatalf("want base + revision 1 + revision 2, got %v", lines)
	}
	if !strings.Contains(lines[1], `"revision":1`) || !strings.Contains(lines[1], `"refinement_reopens":2`) {
		t.Fatalf("revision 1 line = %s", lines[1])
	}
}

func TestAppendRefinementHandoffLineIsIdempotentByMissionID(t *testing.T) {
	path := handoffLinePath(t)
	first := RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: i64(100)}
	if _, err := AppendRefinementHandoffLine(path, first); err != nil {
		t.Fatal(err)
	}
	appended, err := AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: i64(999)})
	if err != nil || appended {
		t.Fatalf("second append: appended=%v err=%v", appended, err)
	}
	if got := readLines(t, path); len(got) != 1 || !strings.Contains(got[0], `"discovery_tokens":100`) {
		t.Fatalf("history changed: %v", got)
	}
	if appended, err = AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: "m-2"}); err != nil || !appended {
		t.Fatalf("a different mission is appended: appended=%v err=%v", appended, err)
	}
}

func TestRefinementHandoffLineValidation(t *testing.T) {
	cases := map[string]RefinementHandoffLine{
		"mission_id is required":               {},
		"discovery_tokens must be >= 0":        {MissionID: "m", DiscoveryTokens: i64(-1)},
		"brief_tokens must be >= 0":            {MissionID: "m", BriefTokens: i64(-1)},
		"refinement_reopens must be >= 0":      {MissionID: "m", RefinementReopens: intp(-1)},
		"revision must be >= 1":                {MissionID: "m", Revision: intp(0)},
		"brief_compression_ratio must be >= 0": {MissionID: "m", BriefCompressionRatio: f64(-0.1)},
		"evidence_coverage_ratio must be >= 0": {MissionID: "m", EvidenceCoverageRatio: f64(-0.1)},
	}
	for want, line := range cases {
		path := handoffLinePath(t)
		_, err := AppendRefinementHandoffLine(path, line)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q, got %v", want, err)
		}
		if _, statErr := os.Stat(path); statErr == nil {
			t.Fatalf("%q: nothing may be written for an invalid line", want)
		}
	}
}

func appendConcurrently(t *testing.T, path string, ids []string) {
	t.Helper()
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := AppendRefinementHandoffLine(path, RefinementHandoffLine{MissionID: id}); err != nil {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
}

func missionCounts(t *testing.T, path string) map[string]int {
	t.Helper()
	seen := map[string]int{}
	for _, l := range readLines(t, path) {
		var e RefinementHandoffLine
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("corrupt line %q: %v", l, err)
		}
		seen[e.MissionID]++
	}
	return seen
}

func TestAppendRefinementHandoffLineConcurrentAppendsKeepEveryMission(t *testing.T) {
	path := handoffLinePath(t)
	ids := []string{"a", "b", "c", "d", "e", "f"}
	appendConcurrently(t, path, append(append([]string{}, ids...), ids...))
	seen := missionCounts(t, path)
	for _, id := range ids {
		if seen[id] != 1 {
			t.Fatalf("mission %s appears %d times", id, seen[id])
		}
	}
}

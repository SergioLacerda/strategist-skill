package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func validFallbackDecisionLine(missionID, slot, policy, outcome string, userConfirmed bool) string {
	rec := FallbackDecision{
		MissionID:          missionID,
		Slot:               slot,
		Phase:              "refinement",
		Policy:             policy,
		Outcome:            outcome,
		ConfiguredProvider: "openspec-explore",
		EffectiveProvider:  "archivist",
		Reason:             "role_invocation_failed",
		UserConfirmed:      userConfirmed,
		Degraded:           true,
		Timestamp:          "2026-08-20T00:00:00Z",
	}
	line, err := json.Marshal(rec)
	if err != nil {
		panic(err)
	}
	return string(line)
}

func TestValidateFallbackDecisionLine_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{
		validFallbackDecisionLine("m-1", "execution", "native", "auto_native", false),
		validFallbackDecisionLine("m-2", "refinement", "ask", "ask_required", true),
	}
	for _, line := range cases {
		if err := ValidateFallbackDecisionLine(line); err != nil {
			t.Fatalf("unexpected error for valid line %q: %v", line, err)
		}
	}
}

func TestValidateFallbackDecisionLine_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		line    string
		missing string
	}{
		{`{"slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "mission_id"},
		{`{"mission_id":"m-1","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "slot"},
		{`{"mission_id":"m-1","slot":"execution","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "phase"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "policy"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "outcome"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "configured_provider"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "effective_provider"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`, "reason"},
		{`{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true}`, "timestamp"},
	}
	for _, tc := range cases {
		err := ValidateFallbackDecisionLine(tc.line)
		if err == nil {
			t.Fatalf("expected error for missing %q, got nil", tc.missing)
		}
		if !strings.Contains(err.Error(), tc.missing) {
			t.Fatalf("error should mention %q: %v", tc.missing, err)
		}
	}
}

func TestValidateFallbackDecisionLine_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		line string
	}{
		{"bad slot", `{"mission_id":"m-1","slot":"teleport","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"bad policy", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"maybe","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"bad outcome", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"blocked","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"same configured and effective provider", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"a","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"degraded false", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"native","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":false,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"outcome does not match policy table", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"block","outcome":"auto_native","configured_provider":"a","effective_provider":"b","reason":"r","degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
		{"ask outcome without user confirmation", `{"mission_id":"m-1","slot":"execution","phase":"p","policy":"ask","outcome":"ask_required","configured_provider":"a","effective_provider":"b","reason":"r","user_confirmed":false,"degraded":true,"timestamp":"2026-08-20T00:00:00Z"}`},
	}
	for _, tc := range cases {
		if err := ValidateFallbackDecisionLine(tc.line); err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}

func TestValidateFallbackDecisionLine_NotJSON(t *testing.T) {
	t.Parallel()
	if err := ValidateFallbackDecisionLine("not json"); err == nil {
		t.Fatal("expected error for non-JSON line")
	}
}

func TestAppendFallbackDecisionLine_WritesValidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-decisions.jsonl")
	line := validFallbackDecisionLine("m-3", "execution", "native", "auto_native", false)

	appended, err := AppendFallbackDecisionLine(path, line)
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

func TestAppendFallbackDecisionLine_RejectsInvalidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-decisions.jsonl")

	_, err := AppendFallbackDecisionLine(path, `{"mission_id":""}`)
	if err == nil {
		t.Fatal("expected error for invalid line")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("file should not be created on validation failure")
	}
}

func TestAppendFallbackDecisionLine_OpenError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent-subdir", "fallback-decisions.jsonl")
	line := validFallbackDecisionLine("m-err", "execution", "native", "auto_native", false)
	if _, err := AppendFallbackDecisionLine(path, line); err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
}

func TestAppendFallbackDecisionLine_SkipsDuplicateKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-decisions.jsonl")
	line := validFallbackDecisionLine("m-loop", "execution", "native", "auto_native", false)

	for i := range 3 {
		appended, err := AppendFallbackDecisionLine(path, line)
		if err != nil {
			t.Fatalf("append %d: unexpected error: %v", i, err)
		}
		wantAppended := i == 0
		if appended != wantAppended {
			t.Fatalf("append %d: got appended=%v, want %v", i, appended, wantAppended)
		}
	}

	entries := readFallbackDecisionTestEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 line, got %d", len(entries))
	}
}

func TestAppendFallbackDecisionLine_SameMissionDifferentSlotBothAppend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-decisions.jsonl")

	first := validFallbackDecisionLine("m-multi", "refinement", "ask", "ask_required", true)
	second := validFallbackDecisionLine("m-multi", "execution", "native", "auto_native", false)

	for _, line := range []string{first, second} {
		appended, err := AppendFallbackDecisionLine(path, line)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !appended {
			t.Fatalf("expected distinct slot within same mission_id to be appended: %s", line)
		}
	}

	entries := readFallbackDecisionTestEntries(t, path)
	if len(entries) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(entries))
	}
}

func TestReadFallbackDecisions_MissingFileReturnsNilNotError(t *testing.T) {
	t.Parallel()
	records, err := ReadFallbackDecisions(filepath.Join(t.TempDir(), "absent.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records, got %v", records)
	}
}

func TestReadFallbackDecisions_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "fallback-decisions.jsonl")
	line := validFallbackDecisionLine("m-ok", "execution", "native", "auto_native", false)
	content := "not json\n" + line + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, err := ReadFallbackDecisions(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].MissionID != "m-ok" {
		t.Fatalf("unexpected record: %+v", records[0])
	}
}

func TestUnlockFallbackDecisionFile_ReturnsErrorOnClosedFile(t *testing.T) {
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
	unlockFallbackDecisionFile(f, &outerErr)
	if outerErr == nil {
		t.Fatal("expected error unlocking a closed file, got nil")
	}
}

func TestAppendFallbackDecisionLineLocked_LockFailureOnClosedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "lock-closed"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	line := validFallbackDecisionLine("m-lock", "execution", "native", "auto_native", false)
	if _, err := appendFallbackDecisionLineLocked(f, "m-lock", "execution", line); err == nil {
		t.Fatal("expected lock error on closed file, got nil")
	}
}

func TestFallbackDecisionLineHasKey_EmptyLine(t *testing.T) {
	t.Parallel()
	if fallbackDecisionLineHasKey(nil, "m-1", "execution") {
		t.Fatal("expected false for empty line")
	}
}

func TestFallbackDecisionLineHasKey_MalformedJSONTolerated(t *testing.T) {
	t.Parallel()
	if fallbackDecisionLineHasKey([]byte("not json"), "m-1", "execution") {
		t.Fatal("expected false for malformed JSON, not an error")
	}
}

func TestFallbackDecisionHistoryPath(t *testing.T) {
	t.Parallel()
	got := FallbackDecisionHistoryPath(filepath.FromSlash("/tmp/strategist-root"))
	want := filepath.Join(filepath.FromSlash("/tmp/strategist-root"), filepath.FromSlash("memory/fallback-decisions.jsonl"))
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// readFallbackDecisionTestEntries reads and parses every line in path as a
// FallbackDecision, failing the test on any read or parse error.
func readFallbackDecisionTestEntries(t *testing.T, path string) []FallbackDecision {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []FallbackDecision
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var e FallbackDecision
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("parse line %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	return entries
}

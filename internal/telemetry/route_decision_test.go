package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRouteDecisionLine_Valid(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"default","route_confidence":0.9,"evidence_state":"requires_discovery","discovery_subtype":"creative","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`,
		`{"mission_id":"m-2","request_category":"analysis_move","selected_route":"critical_hit","route_reason":"plain move","route_confidence":1.0,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`,
	}
	for _, line := range cases {
		if err := ValidateRouteDecisionLine(line); err != nil {
			t.Fatalf("unexpected error for valid line %q: %v", line, err)
		}
	}
}

func TestValidateRouteDecisionLine_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		line    string
		missing string
	}{
		{`{"request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`, "mission_id"},
		{`{"mission_id":"m-1","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`, "request_category"},
		{`{"mission_id":"m-1","request_category":"general","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`, "selected_route"},
		{`{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`, "route_reason"},
		{`{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`, "evidence_state"},
		{`{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline"}`, "timestamp"},
		{`{}`, "mission_id"},
	}
	for _, tc := range cases {
		err := ValidateRouteDecisionLine(tc.line)
		if err == nil {
			t.Fatalf("expected error for missing %q, got nil", tc.missing)
		}
		if !strings.Contains(err.Error(), tc.missing) {
			t.Fatalf("error should mention %q: %v", tc.missing, err)
		}
	}
}

func TestValidateRouteDecisionLine_InvalidValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		line string
	}{
		{
			name: "bad selected_route",
			line: `{"mission_id":"m-1","request_category":"general","selected_route":"teleport","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`,
		},
		{
			name: "bad evidence_state",
			line: `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"maybe","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`,
		},
		{
			name: "confidence out of range",
			line: `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":1.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`,
		},
		{
			name: "bad fallback_route",
			line: `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"critical_hit","timestamp":"2026-08-03T00:00:00Z"}`,
		},
	}
	for _, tc := range cases {
		if err := ValidateRouteDecisionLine(tc.line); err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}

func TestValidateRouteDecisionLine_NotJSON(t *testing.T) {
	t.Parallel()
	if err := ValidateRouteDecisionLine("not json"); err == nil {
		t.Fatal("expected error for non-JSON line")
	}
}

func TestAppendRouteDecisionLine_WritesValidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "route-decisions.jsonl")
	line := `{"mission_id":"m-3","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.8,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`

	appended, err := AppendRouteDecisionLine(path, line)
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

func TestAppendRouteDecisionLine_RejectsInvalidLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "route-decisions.jsonl")

	_, err := AppendRouteDecisionLine(path, `{"mission_id":""}`)
	if err == nil {
		t.Fatal("expected error for invalid line")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("file should not be created on validation failure")
	}
}

func TestAppendRouteDecisionLine_OpenError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nonexistent-subdir", "route-decisions.jsonl")
	line := `{"mission_id":"m-err","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`
	if _, err := AppendRouteDecisionLine(path, line); err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
}

func TestAppendRouteDecisionLine_SkipsDuplicateMissionID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "route-decisions.jsonl")
	line := `{"mission_id":"m-loop","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`

	for i := range 3 {
		appended, err := AppendRouteDecisionLine(path, line)
		if err != nil {
			t.Fatalf("append %d: unexpected error: %v", i, err)
		}
		wantAppended := i == 0
		if appended != wantAppended {
			t.Fatalf("append %d: got appended=%v, want %v", i, appended, wantAppended)
		}
	}

	entries := readRouteDecisionTestEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 line, got %d", len(entries))
	}
}

func TestAppendRouteDecisionLine_SameRouteDifferentMissionIDBothAppend(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "route-decisions.jsonl")

	first := `{"mission_id":"m-a","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`
	second := `{"mission_id":"m-b","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:01Z"}`

	for _, line := range []string{first, second} {
		appended, err := AppendRouteDecisionLine(path, line)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !appended {
			t.Fatalf("expected distinct mission_id %q to be appended", line)
		}
	}

	entries := readRouteDecisionTestEntries(t, path)
	if len(entries) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(entries))
	}
}

func TestUnlockRouteDecisionFile_ReturnsErrorOnClosedFile(t *testing.T) {
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
	unlockRouteDecisionFile(f, &outerErr)
	if outerErr == nil {
		t.Fatal("expected error unlocking a closed file, got nil")
	}
}

func TestAppendRouteDecisionLineLocked_LockFailureOnClosedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "lock-closed"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	line := `{"mission_id":"m-lock","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`
	if _, err := appendRouteDecisionLineLocked(f, "m-lock", line); err == nil {
		t.Fatal("expected lock error on closed file, got nil")
	}
}

func TestRouteDecisionLineHasMissionID_EmptyLine(t *testing.T) {
	t.Parallel()
	if routeDecisionLineHasMissionID(nil, "m-1") {
		t.Fatal("expected false for empty line")
	}
}

func TestRouteDecisionLineHasMissionID_MalformedJSONTolerated(t *testing.T) {
	t.Parallel()
	if routeDecisionLineHasMissionID([]byte("not json"), "m-1") {
		t.Fatal("expected false for malformed JSON, not an error")
	}
}

func TestAppendRouteDecisionLineLocked_SeekFailurePropagatesFromMissionIDExists(t *testing.T) {
	t.Parallel()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer w.Close()
	defer r.Close()

	line := `{"mission_id":"m-seek","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.5,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`
	_, err = appendRouteDecisionLineLocked(r, "m-seek", line)
	if err == nil {
		t.Fatal("expected error from non-seekable file, got nil")
	}
	if !strings.Contains(err.Error(), "scan route decisions file") {
		t.Fatalf("expected error to be wrapped by routeDecisionMissionIDExists's scan context, got: %v", err)
	}
}

// readRouteDecisionTestEntries reads and parses every line in path as a
// RouteDecision, failing the test on any read or parse error.
func readRouteDecisionTestEntries(t *testing.T, path string) []RouteDecision {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []RouteDecision
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var e RouteDecision
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("parse line %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	return entries
}

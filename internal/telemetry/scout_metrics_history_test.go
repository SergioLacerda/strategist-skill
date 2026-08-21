package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadRouteDecisions_MissingFileIsEmptyNotError(t *testing.T) {
	t.Parallel()
	decisions, err := ReadRouteDecisions(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decisions != nil {
		t.Fatalf("expected nil decisions, got %v", decisions)
	}
}

func TestReadRouteDecisions_ParsesValidLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "route-decisions.jsonl")
	content := `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.8,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}
{"mission_id":"m-2","request_category":"general","selected_route":"critical_hit","route_reason":"r","route_confidence":0.9,"evidence_state":"explicit","fallback_route":"","timestamp":"2026-08-03T00:05:00Z"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	decisions, err := ReadRouteDecisions(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decisions))
	}
	if decisions[0].MissionID != "m-1" || decisions[1].MissionID != "m-2" {
		t.Fatalf("unexpected decisions: %+v", decisions)
	}
}

func TestReadRouteDecisions_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "route-decisions.jsonl")
	content := "not json\n" + `{"mission_id":"m-1","selected_route":"full_pipeline"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	decisions, err := ReadRouteDecisions(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision (malformed line skipped), got %d", len(decisions))
	}
}

func TestReadOutcomes_MissingFileIsEmptyNotError(t *testing.T) {
	t.Parallel()
	outcomes, err := ReadOutcomes(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcomes != nil {
		t.Fatalf("expected nil outcomes, got %v", outcomes)
	}
}

func TestReadOutcomes_ParsesValidLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	content := `{"mission_id":"m-1","status":"analysis_delivered","timestamp":"2026-08-03T00:00:00Z"}
{"mission_id":"m-2","status":"documentation_applied","timestamp":"2026-08-03T00:05:00Z"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	outcomes, err := ReadOutcomes(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(outcomes))
	}
	if outcomes[0].MissionID != "m-1" || outcomes[1].Status != "documentation_applied" {
		t.Fatalf("unexpected outcomes: %+v", outcomes)
	}
}

func TestReadOutcomes_SkipsMalformedLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	content := "not json\n" + `{"mission_id":"m-1","status":"analysis_delivered"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	outcomes, err := ReadOutcomes(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome (malformed line skipped), got %d", len(outcomes))
	}
}

func TestReadRouteDecisions_IntegratesWithComputeRouteMetrics(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	decisionsPath := filepath.Join(dir, "route-decisions.jsonl")
	outcomesPath := filepath.Join(dir, "outcomes.jsonl")

	decisionLine := `{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.8,"evidence_state":"explicit","fallback_route":"full_pipeline","timestamp":"2026-08-03T00:00:00Z"}`
	if _, err := AppendRouteDecisionLine(decisionsPath, decisionLine); err != nil {
		t.Fatalf("append route decision: %v", err)
	}
	outcomeLine := `{"mission_id":"m-1","status":"analysis_delivered","timestamp":"2026-08-03T00:00:00Z"}`
	if _, err := AppendOutcomeLine(outcomesPath, outcomeLine); err != nil {
		t.Fatalf("append outcome: %v", err)
	}

	decisions, err := ReadRouteDecisions(decisionsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outcomes, err := ReadOutcomes(outcomesPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := ComputeRouteMetrics(decisions, outcomes)
	if m.SampleSize != 1 {
		t.Fatalf("SampleSize = %d, want 1", m.SampleSize)
	}
	if m.FallbackRate != 1.0 {
		t.Fatalf("FallbackRate = %v, want 1.0", m.FallbackRate)
	}
}

func TestRouteDecisionHistoryPath(t *testing.T) {
	t.Parallel()
	got := RouteDecisionHistoryPath("/tmp/strategist-root")
	want := filepath.Join("/tmp/strategist-root", "memory", "route-decisions.jsonl")
	if got != want {
		t.Fatalf("unexpected path: got %q, want %q", got, want)
	}
}

func TestOutcomeHistoryPath(t *testing.T) {
	t.Parallel()
	got := OutcomeHistoryPath("/tmp/strategist-root")
	want := filepath.Join("/tmp/strategist-root", "memory", "outcomes.jsonl")
	if got != want {
		t.Fatalf("unexpected path: got %q, want %q", got, want)
	}
}

func TestReadRouteDecisions_OpenError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR open semantics differ on Windows")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	// blocker is a file, so opening blocker/route-decisions.jsonl fails with
	// ENOTDIR — a real error distinct from os.ErrNotExist.
	path := filepath.Join(blocker, "route-decisions.jsonl")

	if _, err := ReadRouteDecisions(path); err == nil {
		t.Fatal("expected error when parent path is not a directory, got nil")
	}
}

func TestReadOutcomes_OpenError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR open semantics differ on Windows")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	path := filepath.Join(blocker, "outcomes.jsonl")

	if _, err := ReadOutcomes(path); err == nil {
		t.Fatal("expected error when parent path is not a directory, got nil")
	}
}

func TestParseRouteDecisionLine_EmptyLine(t *testing.T) {
	t.Parallel()
	if _, ok := parseRouteDecisionLine(nil); ok {
		t.Fatal("expected ok=false for an empty line")
	}
}

func TestParseOutcomeLine_EmptyLine(t *testing.T) {
	t.Parallel()
	if _, ok := parseOutcomeLine(nil); ok {
		t.Fatal("expected ok=false for an empty line")
	}
}

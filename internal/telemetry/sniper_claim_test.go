package telemetry

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendSniperClaim_ThenReadRecentSniperClaims_RoundTrips(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "memory", "sniper-claims.jsonl")
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)

	rec := SniperClaimRecord{MissionID: "m-1", BasePath: ".analysis", TargetPath: "docs/foo.md", ClaimedAt: now}
	if err := AppendSniperClaim(path, rec); err != nil {
		t.Fatalf("AppendSniperClaim: %v", err)
	}

	got, err := ReadRecentSniperClaims(path, now, SniperClaimWindow)
	if err != nil {
		t.Fatalf("ReadRecentSniperClaims: %v", err)
	}
	if len(got) != 1 || got[0].MissionID != "m-1" || got[0].TargetPath != "docs/foo.md" {
		t.Fatalf("unexpected records: %+v", got)
	}
}

func TestReadRecentSniperClaims_MissingFileReturnsNilNil(t *testing.T) {
	t.Parallel()
	got, err := ReadRecentSniperClaims(filepath.Join(t.TempDir(), "missing.jsonl"), time.Now(), SniperClaimWindow)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil records, got %+v", got)
	}
}

func TestReadRecentSniperClaims_SkipsMalformedAndOutOfWindowLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "claims.jsonl")
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	old := now.Add(-SniperClaimWindow - time.Hour)

	lines := []string{
		"not json at all",
		`{"mission_id":"m-old","base_path":".analysis","target_path":"docs/old.md","claimed_at":"` + old.Format(time.RFC3339) + `"}`,
		`{"mission_id":"m-recent","base_path":".analysis","target_path":"docs/recent.md","claimed_at":"` + now.Format(time.RFC3339) + `"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := ReadRecentSniperClaims(path, now, SniperClaimWindow)
	if err != nil {
		t.Fatalf("ReadRecentSniperClaims: %v", err)
	}
	if len(got) != 1 || got[0].MissionID != "m-recent" {
		t.Fatalf("expected only the in-window record, got %+v", got)
	}
}

func TestDetectClaimCollisions_TwoDistinctMissionsSameTargetIsCollision(t *testing.T) {
	t.Parallel()
	now := time.Now()
	records := []SniperClaimRecord{
		{MissionID: "m-1", TargetPath: "docs/foo.md", BasePath: ".analysis", ClaimedAt: now},
		{MissionID: "m-2", TargetPath: "docs/foo.md", BasePath: ".analysis", ClaimedAt: now.Add(time.Minute)},
	}
	signals := DetectClaimCollisions(records)
	if len(signals) != 1 {
		t.Fatalf("expected exactly one collision signal, got %d: %+v", len(signals), signals)
	}
	got := signals[0]
	if got.TargetPath != "docs/foo.md" {
		t.Fatalf("unexpected target path: %q", got.TargetPath)
	}
	if len(got.MissionIDs) != 2 {
		t.Fatalf("expected 2 colliding mission ids, got %+v", got.MissionIDs)
	}
}

func TestDetectClaimCollisions_SameMissionClaimingTwiceIsNotACollision(t *testing.T) {
	t.Parallel()
	now := time.Now()
	records := []SniperClaimRecord{
		{MissionID: "m-1", TargetPath: "docs/foo.md", BasePath: ".analysis", ClaimedAt: now},
		{MissionID: "m-1", TargetPath: "docs/foo.md", BasePath: ".analysis", ClaimedAt: now.Add(time.Minute)},
	}
	signals := DetectClaimCollisions(records)
	if len(signals) != 0 {
		t.Fatalf("expected no collision for repeated claims by the same mission, got %+v", signals)
	}
}

func TestDetectClaimCollisions_DifferentTargetsIsNotACollision(t *testing.T) {
	t.Parallel()
	now := time.Now()
	records := []SniperClaimRecord{
		{MissionID: "m-1", TargetPath: "docs/foo.md", BasePath: ".analysis", ClaimedAt: now},
		{MissionID: "m-2", TargetPath: "docs/bar.md", BasePath: ".analysis", ClaimedAt: now},
	}
	signals := DetectClaimCollisions(records)
	if len(signals) != 0 {
		t.Fatalf("expected no collision for distinct target paths, got %+v", signals)
	}
}

func TestDetectClaimCollisions_EmptyInputReturnsNil(t *testing.T) {
	t.Parallel()
	if got := DetectClaimCollisions(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestClaimCollisionThresholdMet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		count int
		want  bool
	}{
		{0, false},
		{1, false},
		{2, true},
		{3, true},
	}
	for _, c := range cases {
		if got := ClaimCollisionThresholdMet(c.count); got != c.want {
			t.Fatalf("ClaimCollisionThresholdMet(%d) = %v, want %v", c.count, got, c.want)
		}
	}
}

func TestFormatClaimCollisionSignal(t *testing.T) {
	t.Parallel()
	line := FormatClaimCollisionSignal(ClaimCollisionSignal{
		BasePath:   ".analysis",
		TargetPath: "docs/foo.md",
		MissionIDs: []string{"m-1", "m-2"},
	})
	want := "[Strategist] signal=sniper_claim_collision base_path=.analysis target=docs/foo.md missions=m-1,m-2"
	if line != want {
		t.Fatalf("unexpected line\nwant: %s\n got: %s", want, line)
	}
}

func TestEmitClaimCollisionSignal_LogsLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	EmitClaimCollisionSignal(ClaimCollisionSignal{
		BasePath:   ".analysis",
		TargetPath: "docs/foo.md",
		MissionIDs: []string{"m-1", "m-2"},
	})

	if !strings.Contains(buf.String(), "sniper_claim_collision") {
		t.Fatalf("expected log output to contain signal name, got: %s", buf.String())
	}
}

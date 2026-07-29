package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestFormatSniperConflictSignal(t *testing.T) {
	t.Parallel()
	line := FormatSniperConflictSignal(SniperConflictSignal{
		MissionID:     "m-1",
		BasePath:      ".analysis",
		TargetPath:    "docs/runbooks/example.md",
		ConflictCount: 3,
	})
	want := "[Strategist] signal=sniper_conflict_attributed mission=m-1 base_path=.analysis target=docs/runbooks/example.md conflict_count=3"
	if line != want {
		t.Fatalf("unexpected line\nwant: %s\n got: %s", want, line)
	}
}

func TestFormatSniperConflictSignal_SanitizesAbsolutePaths(t *testing.T) {
	t.Parallel()
	line := FormatSniperConflictSignal(SniperConflictSignal{
		MissionID:  "m-1",
		BasePath:   "/home/user/workspace",
		TargetPath: "/home/user/workspace/docs/runbooks/example.md",
	})
	if strings.Contains(line, "/home/user") {
		t.Fatalf("expected absolute paths to be redacted, got: %s", line)
	}
	if !strings.Contains(line, "<redacted-path>") {
		t.Fatalf("expected redacted-path sentinel, got: %s", line)
	}
}

func TestEmitSniperConflictSignal_LogsLine(t *testing.T) {
	// no t.Parallel() — test mutates slog.Default() global state
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	EmitSniperConflictSignal(SniperConflictSignal{
		MissionID:     "m-2",
		BasePath:      ".analysis",
		TargetPath:    "docs/runbooks/example.md",
		ConflictCount: 3,
	})

	_ = h.Handle(context.Background(), slog.Record{})
	out := buf.String()
	if !strings.Contains(out, "signal=sniper_conflict_attributed mission=m-2 base_path=.analysis target=docs/runbooks/example.md conflict_count=3") {
		t.Fatalf("log output missing canonical signal line: %s", out)
	}
	if !strings.Contains(out, AttrConflictCount) || !strings.Contains(out, AttrBasePath) {
		t.Fatalf("log output missing structured signal attrs: %s", out)
	}
}

func TestClassifyConflictedTargets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		conflicted           []string
		recentlyMaterialized []string
		want                 []string
	}{
		{
			name:                 "attributed subset",
			conflicted:           []string{"docs/a.md", "docs/b.md", "internal/unrelated.go"},
			recentlyMaterialized: []string{"docs/a.md", "docs/c.md"},
			want:                 []string{"docs/a.md"},
		},
		{
			name:                 "no overlap",
			conflicted:           []string{"internal/unrelated.go"},
			recentlyMaterialized: []string{"docs/a.md"},
			want:                 nil,
		},
		{
			name:                 "empty inputs",
			conflicted:           nil,
			recentlyMaterialized: nil,
			want:                 nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyConflictedTargets(tt.conflicted, tt.recentlyMaterialized)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("unexpected result\nwant: %v\n got: %v", tt.want, got)
			}
		})
	}
}

func TestF3ConflictThresholdMet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		count int
		want  bool
	}{
		{count: 0, want: false},
		{count: 2, want: false},
		{count: 3, want: true},
		{count: 4, want: true},
	}
	for _, tt := range tests {
		if got := F3ConflictThresholdMet(tt.count); got != tt.want {
			t.Errorf("F3ConflictThresholdMet(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestSniperMaterializationHistory_ReadRecentSkipsMalformedAndOld(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "sniper-materializations.jsonl")

	mustAppendSniperMaterialization(t, path, SniperMaterializationRecord{
		MissionID:      "recent",
		BasePath:       ".analysis",
		TargetPath:     "docs/recent.md",
		MaterializedAt: now.Add(-time.Hour),
	})
	mustAppendSniperMaterialization(t, path, SniperMaterializationRecord{
		MissionID:      "old",
		BasePath:       ".analysis",
		TargetPath:     "docs/old.md",
		MaterializedAt: now.Add(-31 * 24 * time.Hour),
	})
	appendRawHistoryLine(t, path, "not json\n")

	records, err := ReadRecentSniperMaterializations(path, now, SniperMaterializationWindow)
	if err != nil {
		t.Fatalf("read recent: %v", err)
	}
	assertSingleSniperMaterializationRecord(t, records, "docs/recent.md")
}

func mustAppendSniperMaterialization(t *testing.T, path string, rec SniperMaterializationRecord) {
	t.Helper()
	if err := AppendSniperMaterialization(path, rec); err != nil {
		t.Fatalf("append %s: %v", rec.MissionID, err)
	}
}

// appendRawHistoryLine writes line directly to path, bypassing AppendSniperMaterialization's
// JSON marshaling — used to inject a malformed historical record for skip-on-parse-failure tests.
func appendRawHistoryLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open history: %v", err)
	}
	if _, err := f.WriteString(line); err != nil {
		t.Fatalf("write malformed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close history: %v", err)
	}
}

func assertSingleSniperMaterializationRecord(t *testing.T, records []SniperMaterializationRecord, wantTargetPath string) {
	t.Helper()
	if len(records) != 1 || records[0].TargetPath != wantTargetPath {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestAppendSniperMaterialization_MkdirAllFailsWhenParentIsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	// blocker is a file, so MkdirAll(blocker/memory, ...) must fail.
	path := filepath.Join(blocker, "memory", "sniper-materializations.jsonl")

	err := AppendSniperMaterialization(path, SniperMaterializationRecord{MissionID: "m-1", TargetPath: "docs/a.md"})
	if err == nil {
		t.Fatal("expected error when parent dir cannot be created, got nil")
	}
}

func TestAppendSniperMaterialization_OpenFailsWhenPathIsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// path itself is an existing directory — os.OpenFile must fail.
	if err := os.Mkdir(filepath.Join(dir, "history-dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "history-dir")

	err := AppendSniperMaterialization(path, SniperMaterializationRecord{MissionID: "m-1", TargetPath: "docs/a.md"})
	if err == nil {
		t.Fatal("expected error when path is a directory, got nil")
	}
}

func TestSniperConflictSignals_FallsBackToRecordBasePathWhenEmpty(t *testing.T) {
	t.Parallel()
	records := []SniperMaterializationRecord{
		{MissionID: "m-1", BasePath: ".analysis-from-record", TargetPath: "docs/a.md"},
		{MissionID: "m-2", BasePath: ".analysis-from-record", TargetPath: "docs/b.md"},
		{MissionID: "m-3", BasePath: ".analysis-from-record", TargetPath: "docs/c.md"},
	}
	signals := SniperConflictSignals("", []string{"docs/a.md", "docs/b.md", "docs/c.md"}, records)
	if len(signals) != 3 {
		t.Fatalf("expected 3 signals, got %#v", signals)
	}
	for _, signal := range signals {
		if signal.BasePath != ".analysis-from-record" {
			t.Fatalf("expected fallback to record's BasePath, got %q", signal.BasePath)
		}
	}
}

func TestSniperConflictSignals_EmptyInputsReturnNil(t *testing.T) {
	t.Parallel()
	if got := SniperConflictSignals(".analysis", nil, nil); got != nil {
		t.Fatalf("expected nil for empty inputs, got %#v", got)
	}
}

func TestSniperConflictSignals_Threshold(t *testing.T) {
	t.Parallel()
	records := []SniperMaterializationRecord{
		{MissionID: "m-1", BasePath: ".analysis", TargetPath: "docs/a.md"},
		{MissionID: "m-2", BasePath: ".analysis", TargetPath: "docs/b.md"},
		{MissionID: "m-3", BasePath: ".analysis", TargetPath: "docs/c.md"},
	}
	below := SniperConflictSignals(".analysis", []string{"docs/a.md", "docs/b.md"}, records)
	if len(below) != 0 {
		t.Fatalf("expected no signal below threshold, got %#v", below)
	}

	signals := SniperConflictSignals(".analysis", []string{"docs/a.md", "docs/b.md", "docs/c.md"}, records)
	if len(signals) != 3 {
		t.Fatalf("expected 3 signals at threshold, got %#v", signals)
	}
	for _, signal := range signals {
		if signal.ConflictCount != 3 {
			t.Fatalf("expected conflict count 3, got %#v", signal)
		}
	}
}

func TestSniperMaterializationHistoryPath(t *testing.T) {
	t.Parallel()
	got := SniperMaterializationHistoryPath("/tmp/strategist-root")
	want := filepath.Join("/tmp/strategist-root", "memory", "sniper-materializations.jsonl")
	if got != want {
		t.Fatalf("unexpected path: got %q, want %q", got, want)
	}
}

func TestReadRecentSniperMaterializations_MissingFileReturnsNilNil(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "never-created.jsonl")

	records, err := ReadRecentSniperMaterializations(path, time.Now(), SniperMaterializationWindow)
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records for missing file, got %#v", records)
	}
}

func TestReadRecentSniperMaterializations_OpenError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "locked.jsonl")
	if err := os.WriteFile(path, []byte(`{"mission_id":"m"}`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	_, err := ReadRecentSniperMaterializations(path, time.Now(), SniperMaterializationWindow)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestSniperConflictSignals_DeduplicatesSameTargetPath(t *testing.T) {
	t.Parallel()
	records := []SniperMaterializationRecord{
		{MissionID: "m-1-first", BasePath: ".analysis", TargetPath: "docs/a.md"},
		{MissionID: "m-1-second-should-be-ignored", BasePath: ".analysis", TargetPath: "docs/a.md"},
		{MissionID: "m-2", BasePath: ".analysis", TargetPath: "docs/b.md"},
		{MissionID: "m-3", BasePath: ".analysis", TargetPath: "docs/c.md"},
	}

	signals := SniperConflictSignals(".analysis", []string{"docs/a.md", "docs/b.md", "docs/c.md"}, records)
	if len(signals) != 3 {
		t.Fatalf("expected 3 signals (one per distinct target), got %#v", signals)
	}

	var aSignal *SniperConflictSignal
	for i := range signals {
		if signals[i].TargetPath == "docs/a.md" {
			aSignal = &signals[i]
		}
	}
	if aSignal == nil {
		t.Fatal("expected a signal for docs/a.md")
	}
	if aSignal.MissionID != "m-1-first" {
		t.Fatalf("expected first-seen record to win dedup, got mission_id %q", aSignal.MissionID)
	}
}

package main

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func TestParseGitPathLines(t *testing.T) {
	t.Parallel()
	got := parseGitPathLines("\n docs/a.md \n./docs/../docs/b.md\n\n")
	want := []string{"docs/a.md", "docs/b.md"}
	if len(got) != len(want) {
		t.Fatalf("unexpected paths\nwant: %v\n got: %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected paths\nwant: %v\n got: %v", want, got)
		}
	}
}

func TestReadGitConflictedPathsFromWorktree_CommandErrorWrapped(t *testing.T) {
	t.Parallel()
	_, err := readGitConflictedPathsFromWorktree(string([]byte{0}))
	if err == nil {
		t.Fatal("expected command error for invalid worktree")
	}
	if !strings.Contains(err.Error(), "read git conflicted paths") {
		t.Fatalf("expected wrapped context, got %v", err)
	}
}

func TestEmitF3ConflictAttributionSignals_EmitsAtThreshold(t *testing.T) {
	// no t.Parallel() — mutates package global and slog default
	root := filepath.Join(t.TempDir(), ".strategist")
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	historyPath := telemetry.SniperMaterializationHistoryPath(root)
	for _, rec := range []telemetry.SniperMaterializationRecord{
		{MissionID: "m-1", BasePath: ".analysis", TargetPath: "docs/a.md", MaterializedAt: now.Add(-time.Hour)},
		{MissionID: "m-2", BasePath: ".analysis", TargetPath: "docs/b.md", MaterializedAt: now.Add(-time.Hour)},
		{MissionID: "m-3", BasePath: ".analysis", TargetPath: "docs/c.md", MaterializedAt: now.Add(-time.Hour)},
	} {
		if err := telemetry.AppendSniperMaterialization(historyPath, rec); err != nil {
			t.Fatalf("append materialization: %v", err)
		}
	}

	origReader := readGitConflictedPaths
	readGitConflictedPaths = func(string) ([]string, error) {
		return []string{"docs/a.md", "docs/b.md", "docs/c.md"}, nil
	}
	t.Cleanup(func() { readGitConflictedPaths = origReader })

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	if err := emitF3ConflictAttributionSignals(root, ".analysis", now); err != nil {
		t.Fatalf("emit f3 signals: %v", err)
	}
	out := buf.String()
	if count := strings.Count(out, "signal=sniper_conflict_attributed"); count != 3 {
		t.Fatalf("expected 3 emitted signals, got %d: %s", count, out)
	}
}

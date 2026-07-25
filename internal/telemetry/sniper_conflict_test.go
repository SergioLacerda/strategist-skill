package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
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
			if len(got) != len(tt.want) {
				t.Fatalf("unexpected result\nwant: %v\n got: %v", tt.want, got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("unexpected result\nwant: %v\n got: %v", tt.want, got)
				}
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

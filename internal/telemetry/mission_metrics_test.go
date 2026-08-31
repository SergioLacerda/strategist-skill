package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatMissionMetrics(t *testing.T) {
	t.Parallel()

	line := FormatMissionMetrics(MissionMetrics{
		MissionID:         "m-42",
		TStartToIntakeMS:  12,
		TIntakeToRangerMS: 34,
		TotalWallTimeMS:   56,
		TokensIn:          78,
		TokensOut:         90,
		LinesEmitted:      11,
	})

	want := "[Strategist] metrics mission=m-42 t_start_to_intake_ms=12 t_intake_to_scout_ms=0 t_scout_to_ranger_ms=0 t_intake_to_ranger_ms=34 t_ranger_to_archivist_ms=0 t_archivist_to_gate_ms=0 t_gate_wait_ms=0 t_gate_to_sniper_ms=0 t_sniper_to_done_ms=0 total_wall_time_ms=56 tokens_in=78 tokens_out=90 lines_emitted=11"
	if line != want {
		t.Fatalf("unexpected mission metrics line\nwant: %s\n got: %s", want, line)
	}
}

func TestEmitMissionMetrics_LogsLine(t *testing.T) {
	// no t.Parallel() — test mutates slog.Default() global state
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	EmitMissionMetrics(MissionMetrics{
		MissionID:         "m-99",
		TStartToIntakeMS:  1,
		TIntakeToRangerMS: 2,
		TotalWallTimeMS:   3,
		TokensIn:          4,
		TokensOut:         5,
		LinesEmitted:      6,
	})

	_ = h.Handle(context.Background(), slog.Record{})
	out := buf.String()
	if !strings.Contains(out, "[Strategist] metrics mission=m-99 t_start_to_intake_ms=1 t_intake_to_scout_ms=0 t_scout_to_ranger_ms=0 t_intake_to_ranger_ms=2 t_ranger_to_archivist_ms=0 t_archivist_to_gate_ms=0 t_gate_wait_ms=0 t_gate_to_sniper_ms=0 t_sniper_to_done_ms=0 total_wall_time_ms=3 tokens_in=4 tokens_out=5 lines_emitted=6") {
		t.Fatalf("log output missing canonical mission metrics line: %s", out)
	}
}

func TestValidateMissionTokenUsage(t *testing.T) {
	t.Parallel()

	valid := MissionTokenUsageRecord{
		MissionID: "m-1", TokensIn: 10, TokensOut: 20,
		Source: MissionUsageSourceAgentReport, ReportedAt: "2026-08-30T00:00:00Z",
	}
	if err := ValidateMissionTokenUsage(valid); err != nil {
		t.Fatalf("expected valid record to pass, got: %v", err)
	}

	cases := []struct {
		name string
		rec  MissionTokenUsageRecord
	}{
		{"missing mission_id", MissionTokenUsageRecord{TokensIn: 1, TokensOut: 1, Source: "x", ReportedAt: "t"}},
		{"negative tokens_in", MissionTokenUsageRecord{MissionID: "m-1", TokensIn: -1, TokensOut: 1, Source: "x", ReportedAt: "t"}},
		{"negative tokens_out", MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 1, TokensOut: -1, Source: "x", ReportedAt: "t"}},
		{"missing source", MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 1, TokensOut: 1, ReportedAt: "t"}},
		{"missing reported_at", MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 1, TokensOut: 1, Source: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateMissionTokenUsage(tc.rec); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestAppendAndReadMissionTokenUsage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "memory", "mission-token-usage.jsonl")

	rec1 := MissionTokenUsageRecord{
		MissionID: "m-alpha", TokensIn: 111, TokensOut: 222,
		Source: MissionUsageSourceAgentReport, ReportedAt: "2026-08-30T10:00:00Z",
	}
	rec2 := MissionTokenUsageRecord{
		MissionID: "m-beta", TokensIn: 333, TokensOut: 444,
		Source: MissionUsageSourceAgentReport, ReportedAt: "2026-08-30T11:00:00Z",
	}

	if err := AppendMissionTokenUsage(path, rec1); err != nil {
		t.Fatalf("append rec1: %v", err)
	}
	if err := AppendMissionTokenUsage(path, rec2); err != nil {
		t.Fatalf("append rec2: %v", err)
	}

	records, err := ReadMissionTokenUsage(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d: %+v", len(records), records)
	}
	if records[0] != rec1 || records[1] != rec2 {
		t.Fatalf("records do not match what was appended: got %+v", records)
	}

	// The reported numbers must be exactly what was passed in — not zero,
	// not a self-reported placeholder.
	if records[0].TokensIn != 111 || records[0].TokensOut != 222 {
		t.Fatalf("rec1 token counts corrupted on round-trip: %+v", records[0])
	}
}

func TestAppendMissionTokenUsage_RejectsInvalidRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "mission-token-usage.jsonl")

	err := AppendMissionTokenUsage(path, MissionTokenUsageRecord{TokensIn: -5, TokensOut: 1, Source: "x", ReportedAt: "t"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Fatal("file must not be created when the record fails validation")
	}
}

func TestReadMissionTokenUsage_MissingFileIsNotError(t *testing.T) {
	t.Parallel()

	records, err := ReadMissionTokenUsage(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records, got: %+v", records)
	}
}

func TestSetTokens_FeedsMissionTokenUsageRecord(t *testing.T) {
	t.Parallel()

	// Exercises the exact production call path described in SetTokens's doc
	// comment: a MissionRun scoped to a real mission_id, fed explicit
	// caller-reported counts, snapshotted, and turned into a persisted
	// record — mirroring cmd/strategist/mission_report_usage.go's RunE body.
	run := NewMissionRun("m-report-usage")
	run.SetTokens(4096, 1024)
	snap := run.Snapshot()

	rec := MissionTokenUsageRecord{
		MissionID:  snap.MissionID,
		TokensIn:   snap.TokensIn,
		TokensOut:  snap.TokensOut,
		Source:     MissionUsageSourceAgentReport,
		ReportedAt: "2026-08-30T12:00:00Z",
	}
	path := filepath.Join(t.TempDir(), "mission-token-usage.jsonl")
	if err := AppendMissionTokenUsage(path, rec); err != nil {
		t.Fatalf("append: %v", err)
	}

	records, err := ReadMissionTokenUsage(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(records) != 1 || records[0].TokensIn != 4096 || records[0].TokensOut != 1024 {
		t.Fatalf("expected persisted record with SetTokens's values, got: %+v", records)
	}
	if records[0].MissionID != "m-report-usage" {
		t.Fatalf("expected mission_id to carry through, got: %+v", records[0])
	}
}

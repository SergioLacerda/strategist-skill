package telemetry

import (
	"bytes"
	"context"
	"log/slog"
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

	want := "[Strategist] metrics mission=m-42 t_start_to_intake_ms=12 t_intake_to_ranger_ms=34 t_ranger_to_archivist_ms=0 t_archivist_to_gate_ms=0 t_gate_wait_ms=0 t_gate_to_sniper_ms=0 t_sniper_to_done_ms=0 total_wall_time_ms=56 tokens_in=78 tokens_out=90 lines_emitted=11"
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
	if !strings.Contains(out, "[Strategist] metrics mission=m-99 t_start_to_intake_ms=1 t_intake_to_ranger_ms=2 t_ranger_to_archivist_ms=0 t_archivist_to_gate_ms=0 t_gate_wait_ms=0 t_gate_to_sniper_ms=0 t_sniper_to_done_ms=0 total_wall_time_ms=3 tokens_in=4 tokens_out=5 lines_emitted=6") {
		t.Fatalf("log output missing canonical mission metrics line: %s", out)
	}
}

package telemetry

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMissionRun_SnapshotAndContext(t *testing.T) {
	t.Parallel()

	run := NewMissionRun("m-1")
	ctx := WithMissionRun(context.Background(), run)
	got := MissionRunFromContext(ctx)
	if got == nil {
		t.Fatal("expected mission run in context")
	}

	run.MarkIntake()
	time.Sleep(1 * time.Millisecond)
	run.MarkRanger()
	run.MarkArchivist()
	run.MarkGatePresented()
	time.Sleep(1 * time.Millisecond)
	run.MarkGateResponse()
	run.MarkSniper()
	run.AddLines(3)
	run.SetTokens(11, 22)

	snap := run.Snapshot()
	if snap.MissionID != "m-1" {
		t.Fatalf("unexpected mission id: %s", snap.MissionID)
	}
	allTimings := []int64{
		snap.TStartToIntakeMS, snap.TIntakeToRangerMS, snap.TRangerToArchivistMS,
		snap.TArchivistToGateMS, snap.TGateWaitMS, snap.TGateToSniperMS,
		snap.TSniperToDoneMS, snap.TotalWallTimeMS,
	}
	for _, v := range allTimings {
		if v < 0 {
			t.Fatalf("expected non-negative timings: %#v", snap)
		}
	}
	if snap.TGateWaitMS < 1 {
		t.Fatalf("gate wait must reflect sleep: got %d ms", snap.TGateWaitMS)
	}
	if snap.LinesEmitted != 3 {
		t.Fatalf("unexpected lines emitted: %d", snap.LinesEmitted)
	}
	if snap.TokensIn != 11 || snap.TokensOut != 22 {
		t.Fatalf("unexpected tokens: %#v", snap)
	}
}

func TestMissionRun_PhaseMarks_Idempotent(t *testing.T) {
	t.Parallel()
	run := NewMissionRun("m-idem")
	run.MarkArchivist()
	run.MarkArchivist() // second call must be a no-op
	run.MarkGatePresented()
	run.MarkGatePresented()
	run.MarkGateResponse()
	run.MarkGateResponse()
	run.MarkSniper()
	run.MarkSniper()
	snap := run.Snapshot()
	for _, v := range []int64{snap.TRangerToArchivistMS, snap.TArchivistToGateMS, snap.TGateWaitMS, snap.TGateToSniperMS} {
		if v < 0 {
			t.Fatalf("expected non-negative phase timing: %#v", snap)
		}
	}
}

func TestMissionRunFromContext_NilCtx(t *testing.T) {
	t.Parallel()
	if got := MissionRunFromContext(nil); got != nil { //nolint:staticcheck // intentionally testing the nil-ctx guard
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestMissionRunFromContext_NoKey(t *testing.T) {
	t.Parallel()
	if got := MissionRunFromContext(context.Background()); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestAddLines_NonPositive(t *testing.T) {
	t.Parallel()
	run := NewMissionRun("m-noop")
	run.AddLines(0)
	run.AddLines(-1)
	if snap := run.Snapshot(); snap.LinesEmitted != 0 {
		t.Fatalf("expected 0 lines, got %d", snap.LinesEmitted)
	}
}

func TestSnapshot_ZeroTimestamps(t *testing.T) {
	t.Parallel()
	run := NewMissionRun("m-zero")
	snap := run.Snapshot()
	if snap.TStartToIntakeMS < 0 || snap.TIntakeToRangerMS < 0 || snap.TotalWallTimeMS < 0 {
		t.Fatalf("unexpected negative timings: %#v", snap)
	}
}

func TestFinishMission_NilRun(t *testing.T) {
	t.Parallel()
	FinishMission(context.Background())
}

func TestMissionRun_StartLine(t *testing.T) {
	t.Parallel()
	run := NewMissionRun("m-start")
	line := run.StartLine("local", "/profile", "/active.yaml", "epic", "local_default", "default")
	if !strings.Contains(line, "mission_id=m-start") {
		t.Fatalf("unexpected start line: %s", line)
	}
	if !strings.Contains(line, "profile_mode=local") {
		t.Fatalf("missing profile_mode in start line: %s", line)
	}
}

func TestMissionRun_Finish(t *testing.T) {
	// no t.Parallel() — Finish calls EmitMissionMetrics which uses slog.Info (global state)
	run := NewMissionRun("m-fin")
	run.MarkIntake()
	run.MarkRanger()
	run.Finish()
	if snap := run.Snapshot(); snap.LinesEmitted != 1 {
		t.Fatalf("Finish must call AddLines(1): got %d lines emitted", snap.LinesEmitted)
	}
}

func TestFinishMission_WithRun(t *testing.T) {
	// no t.Parallel() — calls EmitMissionMetrics via slog.Info (global state)
	run := NewMissionRun("m-ctx-fin")
	ctx := WithMissionRun(context.Background(), run)
	FinishMission(ctx)
	if snap := run.Snapshot(); snap.LinesEmitted != 1 {
		t.Fatalf("FinishMission must call AddLines(1): got %d lines emitted", snap.LinesEmitted)
	}
}

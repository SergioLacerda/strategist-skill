package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestFormatPolicyEvent_WithoutReason(t *testing.T) {
	t.Parallel()
	ev := PolicyEvent{
		Phase:           "policy_bootstrap",
		Status:          "done",
		Mission:         "m-1",
		TransitionGroup: "finalize_analysis",
		CorrelationID:   "corr-1",
	}
	line := FormatPolicyEvent(ev)
	want := "[Strategist] phase=policy_bootstrap status=done mission=m-1 documentation_scope=approved_targets transition_group=finalize_analysis correlation_id=corr-1"
	if line != want {
		t.Fatalf("unexpected line\nwant: %s\n got: %s", want, line)
	}
}

func TestFormatPolicyEvent_WithReason(t *testing.T) {
	t.Parallel()
	ev := PolicyEvent{
		Phase:   "documentation_materialization",
		Status:  "skipped",
		Mission: "m-2",
		Reason:  "policy_blocked",
	}
	line := FormatPolicyEvent(ev)
	if !strings.Contains(line, "reason=policy_blocked") {
		t.Fatalf("expected reason field in line: %s", line)
	}
}

func TestEmitPolicyEvent_LogsLine(t *testing.T) {
	// no t.Parallel() — test mutates slog.Default() global state
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	prev := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(prev) })

	EmitPolicyEvent(PolicyEvent{
		Phase:           "policy_eval",
		Status:          "blocked",
		Mission:         "m-3",
		TransitionGroup: "review_gate",
		CorrelationID:   "corr-3",
		Reason:          "policy_blocked",
	})

	_ = h.Handle(context.Background(), slog.Record{})
	out := buf.String()
	if !strings.Contains(out, "[Strategist] phase=policy_eval status=blocked mission=m-3 documentation_scope=approved_targets transition_group=review_gate correlation_id=corr-3 reason=policy_blocked") {
		t.Fatalf("log output missing canonical policy event: %s", out)
	}
	if !strings.Contains(out, AttrTransitionGroup) || !strings.Contains(out, AttrCorrelationID) {
		t.Fatalf("log output missing structured policy attrs: %s", out)
	}
}

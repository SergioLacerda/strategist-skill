package telemetryverify

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lc(id, kind, phase, run, reason, at string) telemetry.MissionLifecycleRecord {
	return telemetry.MissionLifecycleRecord{
		SchemaVersion: "1", MarkerID: id, MarkerKind: kind, MissionID: "m-1", Phase: phase,
		PhaseRunID: run, EndReason: reason, ObservedAt: at,
	}
}

func n(v int64) *int64 { return &v }

func us(id, phase, inv string, in, out *int64) telemetry.MissionUsageRecord {
	return telemetry.MissionUsageRecord{
		SchemaVersion: "1", RecordID: id, MissionID: "m-1", Phase: phase, Role: "ranger", InvocationID: inv,
		Attempt: 1, UsageScope: "invocation", RecordKind: "incremental", TokensInput: in, TokensOutput: out,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:10:00Z",
	}
}

func healthy() Input {
	return Input{
		Status: domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseDone, State: domain.StateDoneDelivery},
		Lifecycle: []telemetry.MissionLifecycleRecord{
			lc("a", "mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
			lc("b", "phase_started", "DISCOVERY", "r1", "", "2026-10-10T10:02:00Z"),
			lc("c", "phase_ended", "DISCOVERY", "r1", "completed", "2026-10-10T10:24:00Z"),
			lc("d", "mission_ended", "DONE", "", "completed", "2026-10-10T10:25:00Z"),
		},
		Usage: []telemetry.MissionUsageRecord{us("u1", "DISCOVERY", "inv_1", n(100), n(10))},
	}
}

func codes(r Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func findCode(r Report, code string) *Finding {
	for i := range r.Findings {
		if r.Findings[i].Code == code {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestVerifyHealthyEvidenceIsOK(t *testing.T) {
	r := Verify(healthy())
	assert.True(t, r.OK)
	assert.Empty(t, r.Findings)
	assert.Equal(t, 4, r.Checked.LifecycleRecords)
	assert.Equal(t, 1, r.Checked.UsageRecords)
	assert.NotNil(t, r.Findings, "a JSON consumer gets [] and never null")
}

func TestVerifyLifecycleFindings(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Input)
		want   string
		sev    Severity
	}{
		"invalid record": {func(in *Input) { in.Lifecycle[1].Phase = "DELIVERY" }, "lifecycle_record_invalid", SeverityError},
		"duplicate marker": {func(in *Input) {
			in.Lifecycle = append(in.Lifecycle, in.Lifecycle[1])
		}, "lifecycle_duplicate_marker", SeverityError},
		"time regression": {func(in *Input) { in.Lifecycle[2].ObservedAt = "2026-10-10T09:00:00Z" }, "lifecycle_time_regression", SeverityWarning},
		"end without start": {func(in *Input) {
			in.Lifecycle[1] = lc("b2", "phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:02:00Z")
		}, "lifecycle_end_without_start", SeverityError},
		"end before start": {func(in *Input) { in.Lifecycle[2].ObservedAt = "2026-10-10T10:01:00Z" }, "lifecycle_end_before_start", SeverityError},
		"overlapping run": {func(in *Input) {
			in.Lifecycle = []telemetry.MissionLifecycleRecord{
				lc("a", "phase_started", "DISCOVERY", "r1", "", "2026-10-10T10:00:00Z"),
				lc("b", "phase_started", "DISCOVERY", "r2", "", "2026-10-10T10:05:00Z"),
			}
			in.Status.Phase = domain.PhaseDiscovery
		}, "lifecycle_overlapping_run", SeverityError},
		"run id gap": {func(in *Input) {
			in.Lifecycle[1].PhaseRunID = "r3"
			in.Lifecycle[2].PhaseRunID = "r3"
		}, "lifecycle_run_id_not_contiguous", SeverityWarning},
		"open run at mission end": {func(in *Input) {
			in.Lifecycle = []telemetry.MissionLifecycleRecord{
				lc("b", "phase_started", "DISCOVERY", "r1", "", "2026-10-10T10:02:00Z"),
				lc("d", "mission_ended", "DONE", "", "completed", "2026-10-10T10:25:00Z"),
			}
		}, "lifecycle_open_run_at_mission_end", SeverityError},
		"engine done without end marker": {func(in *Input) { in.Lifecycle = in.Lifecycle[:3] }, "lifecycle_engine_mismatch", SeverityWarning},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := healthy()
			tc.mutate(&in)
			r := Verify(in)
			found := findCode(r, tc.want)
			require.NotNil(t, found, "want %s, got %v", tc.want, codes(r))
			assert.Equal(t, tc.sev, found.Severity)
		})
	}
}

func TestVerifyUsageFindings(t *testing.T) {
	in := healthy()
	dup := us("u1", "DISCOVERY", "inv_1", n(100), n(10))
	conflict := us("u1", "DISCOVERY", "inv_1", n(999), n(10))
	unknownInv := us("u2", "DISCOVERY", "inv_9", n(1), n(1))
	neverStarted := us("u3", "EXECUTION", "inv_1", n(1), n(1))
	unkeyed := us("raw|1", "UNATTRIBUTED", "", n(1), n(1))
	unkeyed.DedupUnavailable = true
	bad := us("u4", "DISCOVERY", "inv_1", n(-1), nil)
	in.KnownInvocations = []string{"inv_1"}
	in.Usage = append(in.Usage, dup, conflict, unknownInv, neverStarted, unkeyed, bad)

	r := Verify(in)
	got := codes(r)
	for _, want := range []string{"usage_duplicate_record", "usage_conflicting_counts", "usage_unknown_invocation",
		"usage_phase_never_started", "usage_dedup_unavailable", "usage_unattributed", "usage_record_invalid"} {
		assert.Contains(t, got, want)
	}
	assert.False(t, r.OK)
}

func TestVerifyCumulativeCounterResetIsAWarning(t *testing.T) {
	in := healthy()
	mk := func(id string, v int64) telemetry.MissionUsageRecord {
		r := us(id, "DISCOVERY", "", n(v), n(1))
		r.RecordKind, r.SessionID, r.UsageScope = "cumulative_snapshot", "s1", "session"
		return r
	}
	in.Usage = []telemetry.MissionUsageRecord{mk("s1a", 55000), mk("s1b", 20000)}
	r := Verify(in)
	assert.Contains(t, codes(r), "usage_counter_reset")
	assert.True(t, r.OK, "a reset is a warning: the interval is unavailable, nothing contradicts itself")
}

func TestVerifyUnreadableLinesAreReported(t *testing.T) {
	in := healthy()
	in.LifecycleMalformed, in.UsageMalformed, in.EventsMalformed = 2, 1, 3
	r := Verify(in)
	assert.Equal(t, 3, r.Counts["warning"])
	assert.Equal(t, []string{"unreadable_lines", "unreadable_lines", "unreadable_lines"}, codes(r))
}

func TestVerifyEventStreamCrossCheck(t *testing.T) {
	in := healthy()
	var events []telemetry.Event
	for _, r := range in.Lifecycle {
		e, err := telemetry.NewMissionLifecycleEvent(r)
		require.NoError(t, err)
		events = append(events, e)
	}
	events = append(events, telemetry.NewMissionUsageEvent(in.Usage[0]))
	in.Events, in.EventsProvided = events, true
	require.True(t, Verify(in).OK)
	assert.Empty(t, Verify(in).Findings, "a complete, consistent stream raises nothing")

	t.Run("missing event", func(t *testing.T) {
		x := in
		x.Events = events[1:]
		assert.Contains(t, codes(Verify(x)), "event_missing_for_record")
	})
	t.Run("duplicate event", func(t *testing.T) {
		x := in
		x.Events = append(append([]telemetry.Event(nil), events...), events[0])
		assert.Contains(t, codes(Verify(x)), "event_duplicate")
	})
	t.Run("event without a record", func(t *testing.T) {
		x := in
		x.Lifecycle = in.Lifecycle[:3]
		x.Status.Phase = domain.PhaseExecution
		r := Verify(x)
		assert.Contains(t, codes(r), "event_without_record")
		assert.False(t, r.OK)
	})
	t.Run("conflicting counts", func(t *testing.T) {
		x := in
		changed := in.Usage[0]
		changed.TokensInput = n(7777)
		x.Events = append(append([]telemetry.Event(nil), events[:4]...), telemetry.NewMissionUsageEvent(changed))
		r := Verify(x)
		assert.Contains(t, codes(r), "event_conflicts_with_record")
		assert.False(t, r.OK)
	})
	t.Run("other mission events are ignored", func(t *testing.T) {
		x := in
		other := in.Usage[0]
		other.MissionID = "m-other"
		x.Events = append(append([]telemetry.Event(nil), events...), telemetry.NewMissionUsageEvent(other))
		assert.Empty(t, Verify(x).Findings)
	})
}

func TestVerifyFindingsAreOrderedBySeverity(t *testing.T) {
	in := healthy()
	in.Usage = append(in.Usage, us("raw|1", "UNATTRIBUTED", "", n(1), n(1)), us("u1", "DISCOVERY", "inv_1", n(5), n(5)))
	in.Usage[1].DedupUnavailable = true
	r := Verify(in)
	require.GreaterOrEqual(t, len(r.Findings), 2)
	assert.Equal(t, SeverityError, r.Findings[0].Severity)
	assert.Equal(t, SeverityInfo, r.Findings[len(r.Findings)-1].Severity)
}

func TestVerifyConflictingDuplicateEventIsAnError(t *testing.T) {
	in := healthy()
	changed := in.Usage[0]
	changed.TokensInput = n(5)
	in.Events = []telemetry.Event{telemetry.NewMissionUsageEvent(in.Usage[0]), telemetry.NewMissionUsageEvent(changed)}
	in.EventsProvided = true
	r := Verify(in)
	assert.Contains(t, codes(r), "event_conflicting_duplicate")
	assert.False(t, r.OK)
}

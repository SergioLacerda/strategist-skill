package usagesummary

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func n(v int64) *int64 { return &v }

type ur = telemetry.MissionUsageRecord

func inc(id, phase, inv string, attempt int, in, out *int64, at string) ur {
	return ur{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: id, MissionID: "m-1",
		Phase: phase, Role: "ranger", InvocationID: inv, Attempt: attempt, UsageScope: "invocation",
		RecordKind: "incremental", TokensInput: in, TokensOutput: out,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: at,
	}
}

func snap(id, session, phase string, in, out *int64, at string) ur {
	r := inc(id, phase, "", 1, in, out, at)
	r.RecordKind, r.SessionID, r.UsageScope = "cumulative_snapshot", session, "session"
	return r
}

func bucket(t *testing.T, s Summary, phase string) Bucket {
	t.Helper()
	b, ok := s.ByPhase[phase]
	require.True(t, ok, "no usage for %s", phase)
	return b
}

func TestSummarizeSumsIncrementalObservationsPerPhase(t *testing.T) {
	s := Summarize([]ur{
		inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(200), "2026-10-10T10:00:00Z"),
		inc("b", "DISCOVERY", "inv_2", 1, n(500), n(100), "2026-10-10T10:05:00Z"),
		inc("c", "REFINEMENT", "inv_3", 1, n(2000), n(400), "2026-10-10T10:30:00Z"),
	}, nil)

	d := bucket(t, s, "DISCOVERY")
	assert.Equal(t, int64(1500), *d.Tokens.Input)
	assert.Equal(t, int64(300), *d.Tokens.Output)
	assert.Equal(t, int64(1800), *d.Tokens.Total)
	assert.Equal(t, 2, d.Observations)
	assert.Equal(t, "reported_unverified", d.Quality)
	assert.Equal(t, int64(3500), *s.Total.Tokens.Input)
	assert.Equal(t, int64(4200), *s.Total.Tokens.Total)
}

func TestSummarizeDeduplicatesTheSameRecordID(t *testing.T) {
	rec := inc("same", "DISCOVERY", "inv_1", 1, n(1000), n(200), "2026-10-10T10:00:00Z")
	s := Summarize([]ur{rec, rec}, nil)
	assert.Equal(t, int64(1000), *bucket(t, s, "DISCOVERY").Tokens.Input, "a replay must not multiply the total")
}

func TestSummarizeRealRetryCountsBothAttempts(t *testing.T) {
	s := Summarize([]ur{
		inc("a1", "REFINEMENT", "inv_1", 1, n(1000), n(100), "2026-10-10T10:00:00Z"),
		inc("a2", "REFINEMENT", "inv_1", 2, n(900), n(90), "2026-10-10T10:10:00Z"),
	}, nil)
	assert.Equal(t, int64(1900), *bucket(t, s, "REFINEMENT").Tokens.Input)
}

func TestSummarizeCumulativeSnapshotsContributeOnlyTheDelta(t *testing.T) {
	s := Summarize([]ur{
		snap("s1", "sess-1", "REFINEMENT", n(40000), n(8000), "2026-10-10T11:00:00Z"),
		snap("s2", "sess-1", "REFINEMENT", n(55000), n(11000), "2026-10-10T11:20:00Z"),
	}, nil)
	b := bucket(t, s, "REFINEMENT")
	assert.Equal(t, int64(15000), *b.Tokens.Input, "55k - 40k, never 95k")
	assert.Equal(t, int64(3000), *b.Tokens.Output)
	assert.Equal(t, 1, s.Coverage.BaselineSnapshots)
}

func TestSummarizeResetAndOtherSessionAreUnavailable(t *testing.T) {
	reset := Summarize([]ur{
		snap("s1", "sess-1", "REFINEMENT", n(55000), n(11000), "2026-10-10T11:00:00Z"),
		snap("s2", "sess-1", "REFINEMENT", n(20000), n(4000), "2026-10-10T11:20:00Z"),
	}, nil)
	_, has := reset.ByPhase["REFINEMENT"]
	assert.False(t, has, "a counter that went backwards yields no interval")
	assert.Equal(t, 1, reset.Coverage.ResetIntervals)
	assert.Nil(t, reset.Total.Tokens.Input)

	other := Summarize([]ur{
		snap("s1", "sess-1", "REFINEMENT", n(40000), n(8000), "2026-10-10T11:00:00Z"),
		snap("s2", "sess-2", "REFINEMENT", n(55000), n(11000), "2026-10-10T11:20:00Z"),
	}, nil)
	assert.Nil(t, other.Total.Tokens.Input, "snapshots of different sessions are never subtracted")
	assert.Equal(t, 2, other.Coverage.BaselineSnapshots)
}

func TestSummarizeUnattributedStaysInTheMissionTotal(t *testing.T) {
	s := Summarize([]ur{
		inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(100), "2026-10-10T10:00:00Z"),
		inc("b", "UNATTRIBUTED", "inv_2", 1, n(300), n(30), "2026-10-10T10:01:00Z"),
	}, nil)
	assert.Equal(t, int64(300), *s.Unattributed.Tokens.Input)
	assert.Equal(t, int64(1300), *s.Total.Tokens.Input)
	_, inPhase := s.ByPhase["UNATTRIBUTED"]
	assert.False(t, inPhase)
}

func TestSummarizeMixedQualityIsNeverPresentedAsOne(t *testing.T) {
	est := inc("b", "DISCOVERY", "inv_2", 1, n(500), n(50), "2026-10-10T10:05:00Z")
	est.MeasurementSource, est.MeasurementQuality = "estimator", "heuristic_estimated"
	s := Summarize([]ur{inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(100), "2026-10-10T10:00:00Z"), est}, nil)

	b := bucket(t, s, "DISCOVERY")
	assert.Equal(t, "mixed", b.Quality)
	assert.Equal(t, map[string]int{"reported_unverified": 1, "heuristic_estimated": 1}, b.QualityBreakdown)
	assert.Equal(t, "mixed", s.Total.Quality)
}

func TestSummarizePartialObservationHasNoTotal(t *testing.T) {
	s := Summarize([]ur{
		inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(100), "2026-10-10T10:00:00Z"),
		inc("b", "DISCOVERY", "inv_2", 1, n(500), nil, "2026-10-10T10:05:00Z"),
	}, nil)
	b := bucket(t, s, "DISCOVERY")
	assert.Equal(t, int64(1500), *b.Tokens.Input)
	assert.Equal(t, int64(100), *b.Tokens.Output)
	assert.Nil(t, b.Tokens.Total, "a total over an incomplete pair would be a guess")
}

func TestSummarizeCacheAndReasoningDoNotChangeTheTotal(t *testing.T) {
	rec := inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(100), "2026-10-10T10:00:00Z")
	rec.TokensCache, rec.TokensReasoning = n(777), n(555)
	s := Summarize([]ur{rec}, nil)
	assert.Equal(t, int64(1100), *bucket(t, s, "DISCOVERY").Tokens.Total)
}

func TestSummarizeNoRecordsIsUnavailableNeverZero(t *testing.T) {
	s := Summarize(nil, nil)
	assert.Nil(t, s.Total.Tokens.Input)
	assert.Nil(t, s.Total.Tokens.Total)
	assert.Equal(t, QualityUnavailable, s.Total.Quality)
	assert.Zero(t, s.Total.Observations)
}

func TestSummarizeCoverageCountsInvocationsWithoutUsage(t *testing.T) {
	s := Summarize(
		[]ur{inc("a", "DISCOVERY", "inv_1", 1, n(10), n(1), "2026-10-10T10:00:00Z")},
		[]string{"inv_1", "inv_2", "inv_2", ""},
	)
	assert.Equal(t, 2, s.Coverage.InvocationsKnown)
	assert.Equal(t, 1, s.Coverage.InvocationsObserved)
	assert.Equal(t, 1, s.Coverage.InvocationsMissing)
}

func TestSummarizeCountsUnkeyedObservations(t *testing.T) {
	rec := inc("raw|m-1|abc", "DISCOVERY", "", 1, n(10), n(1), "2026-10-10T10:00:00Z")
	rec.DedupUnavailable = true
	s := Summarize([]ur{rec}, nil)
	assert.Equal(t, 1, s.Coverage.UnkeyedObservations)
}

func TestSummarizeOrdersSnapshotsByObservedTime(t *testing.T) {
	s := Summarize([]ur{
		snap("late", "sess-1", "REFINEMENT", n(55000), n(1), "2026-10-10T11:20:00Z"),
		snap("early", "sess-1", "REFINEMENT", n(40000), n(1), "2026-10-10T11:00:00Z"),
	}, nil)
	assert.Equal(t, int64(15000), *bucket(t, s, "REFINEMENT").Tokens.Input)
}

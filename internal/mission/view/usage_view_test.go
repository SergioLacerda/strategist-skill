package view

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failAfter struct{ remaining int }

func (f *failAfter) Write(p []byte) (int, error) {
	if f.remaining <= 0 {
		return 0, errors.New("write failed")
	}
	f.remaining--
	return len(p), nil
}

func TestBuildUsageLegacyRowIsOutsideTheTimelineAndTotals(t *testing.T) {
	legacy := []telemetry.MissionTokenUsageRecord{
		{MissionID: "m-1", TokensIn: 100, TokensOut: 20, Source: "agent_report", ReportedAt: "2026-10-10T10:00:00Z"},
		{MissionID: "m-1", TokensIn: 180, TokensOut: 40, Source: "agent_report", ReportedAt: "2026-10-10T11:00:00Z"},
	}
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), TokenUsage: legacy})

	assert.Equal(t, UsageSchemaVersion, v.Schema)
	assert.Equal(t, Available, v.Legacy.Availability)
	assert.Equal(t, 2, v.Legacy.RecordCount)
	assert.Nil(t, v.PhaseTimeline.Total.Tokens.Total, "the legacy ledger is never summed into the total")
	assert.Equal(t, NotApplicable, v.PhaseTimeline.UsageAvailability)
	for _, row := range v.PhaseTimeline.Phases {
		assert.Nil(t, row.Tokens.Input)
		assert.Nil(t, row.Tokens.Total)
	}
}

func TestBuildUsageSharedSectionMatchesTheMissionView(t *testing.T) {
	status := doneStatus()
	view := Build(Input{Status: status, Registry: domain.DefaultRoleRegistry(), Lifecycle: fullPipeline()})
	usage := BuildUsage(UsageInput{Status: status, Lifecycle: fullPipeline()})

	a, err := json.Marshal(view.PhaseTimeline)
	require.NoError(t, err)
	b, err := json.Marshal(usage.PhaseTimeline)
	require.NoError(t, err)
	assert.JSONEq(t, string(a), string(b))
}

func TestBuildUsageDiagnosticsAndAvailability(t *testing.T) {
	v := BuildUsage(UsageInput{
		Status: doneStatus(), LifecycleError: errors.New("a"), TokenUsageError: errors.New("b"),
	})
	assert.Equal(t, Unavailable, v.PhaseTimeline.Availability)
	assert.Equal(t, Unavailable, v.Legacy.Availability)
	require.Len(t, v.Diagnostics, 2)

	none := BuildUsage(UsageInput{Status: doneStatus()})
	assert.Equal(t, NotApplicable, none.Legacy.Availability)
	assert.Empty(t, none.Diagnostics)
}

func TestBuildViewReportsUnreadableLifecycleLedger(t *testing.T) {
	v := Build(Input{Status: doneStatus(), Registry: domain.DefaultRoleRegistry(), LifecycleError: errors.New("x")})
	var reasons []string
	for _, d := range v.Diagnostics {
		reasons = append(reasons, d.Reason)
	}
	assert.Contains(t, reasons, ReasonMissionViewLifecycleUnavailable)
}

func TestRenderUsageHumanShowsDashesRunsAndLegacyDetails(t *testing.T) {
	legacy := []telemetry.MissionTokenUsageRecord{{MissionID: "m-1", TokensIn: 5, TokensOut: 6, Source: "agent_report", ReportedAt: "2026-10-10T10:00:00Z"}}
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), TokenUsage: legacy})

	var short bytes.Buffer
	require.NoError(t, RenderUsageHuman(&short, v, false))
	out := short.String()
	assert.Contains(t, out, "Discovery")
	assert.Contains(t, out, "22m0s")
	assert.Contains(t, out, "MISSION TOTAL   tokens: —")
	assert.Contains(t, out, "1 record(s)")
	assert.NotContains(t, out, "tokens_in=5")
	assert.NotContains(t, out, "run DISCOVERY")

	var detailed bytes.Buffer
	require.NoError(t, RenderUsageHuman(&detailed, v, true))
	assert.Contains(t, detailed.String(), "run DISCOVERY r1")
	assert.Contains(t, detailed.String(), "tokens_in=5 tokens_out=6")
}

func TestRenderUsageHumanWithoutLedgerSaysSo(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, RenderUsageHuman(&out, BuildUsage(UsageInput{Status: doneStatus()}), false))
	assert.Contains(t, out.String(), "no lifecycle recorded")
}

func TestRenderUsageHumanPropagatesWriteErrors(t *testing.T) {
	legacy := []telemetry.MissionTokenUsageRecord{{MissionID: "m-1", Source: "agent_report", ReportedAt: "2026-10-10T10:00:00Z"}}
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), TokenUsage: legacy})
	for budget := 0; budget < 6; budget++ {
		require.Error(t, RenderUsageHuman(&failAfter{remaining: budget}, v, true), "budget %d", budget)
	}
}

func TestBuildUsageAppliesObservedUsageToRowsAndTotal(t *testing.T) {
	usage := []telemetry.MissionUsageRecord{
		inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(200), "2026-10-10T10:10:00Z"),
		inc("b", "REFINEMENT", "inv_2", 1, n(2000), n(400), "2026-10-10T10:40:00Z"),
		inc("c", "UNATTRIBUTED", "", 1, n(50), n(5), "2026-10-10T10:41:00Z"),
	}
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), Usage: usage, KnownInvocations: []string{"inv_1", "inv_2", "inv_3"}})
	tl := v.PhaseTimeline

	assert.Equal(t, Available, tl.UsageAvailability)
	discovery := rowFor(t, tl, "DISCOVERY")
	require.NotNil(t, discovery.Tokens.Total)
	assert.Equal(t, int64(1200), *discovery.Tokens.Total)
	assert.Equal(t, "reported_unverified", discovery.Quality)
	assert.Nil(t, rowFor(t, tl, "EXECUTION").Tokens.Input, "an unobserved phase stays null, never zero")
	assert.Equal(t, "unavailable", rowFor(t, tl, "EXECUTION").Quality)
	assert.Equal(t, "not_applicable", rowFor(t, tl, "APPROVAL_GATE").Quality)
	assert.Equal(t, int64(3655), *tl.Total.Tokens.Total)
	assert.Equal(t, int64(50), *tl.Unattributed.Tokens.Input)
	assert.Equal(t, 1, tl.Coverage.InvocationsMissing)
}

func TestBuildUsageProjectsTokensIntoLifecycleRunDetails(t *testing.T) {
	first := inc("a", "REFINEMENT", "inv-1", 1, n(100), n(10), "2026-10-10T10:30:00Z")
	first.PhaseRunID = "r1"
	second := inc("b", "REFINEMENT", "inv-2", 1, n(200), n(20), "2026-10-10T10:40:00Z")
	second.PhaseRunID = "r2"

	lifecycle := fullPipeline()
	lifecycle = append(lifecycle[:9], append([]telemetry.MissionLifecycleRecord{
		lr("phase_started", "REFINEMENT", "r2", "", "2026-10-10T11:01:00Z"),
		lr("phase_ended", "REFINEMENT", "r2", "completed", "2026-10-10T11:02:00Z"),
	}, lifecycle[9:]...)...)
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: lifecycle, Usage: []telemetry.MissionUsageRecord{first, second}})
	runs := rowFor(t, v.PhaseTimeline, "REFINEMENT").RunDetails
	require.Len(t, runs, 2)
	assert.Equal(t, int64(100), *runs[0].Tokens.Input)
	assert.Equal(t, int64(200), *runs[1].Tokens.Input)
	assert.Equal(t, "reported_unverified", runs[0].Quality)
	assert.Equal(t, int64(300), *rowFor(t, v.PhaseTimeline, "REFINEMENT").Tokens.Input)
}

func TestBuildUsageLeavesUnknownRunUsageOutOfLifecycleDetails(t *testing.T) {
	unknown := inc("unknown", "REFINEMENT", "inv-unknown", 1, n(100), n(10), "2026-10-10T10:30:00Z")
	unknown.PhaseRunID = "missing-run"

	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), Usage: []telemetry.MissionUsageRecord{unknown}})
	runs := rowFor(t, v.PhaseTimeline, "REFINEMENT").RunDetails
	require.Len(t, runs, 1)
	assert.Nil(t, runs[0].Tokens.Input)
	assert.Equal(t, QualityUnavailable, runs[0].Quality)
	assert.Equal(t, int64(100), *rowFor(t, v.PhaseTimeline, "REFINEMENT").Tokens.Input)
}

func TestBuildUsageUsageAppliesEvenWithoutALifecycleLedger(t *testing.T) {
	usage := []telemetry.MissionUsageRecord{inc("a", "DISCOVERY", "inv_1", 1, n(10), n(1), "2026-10-10T10:10:00Z")}
	v := BuildUsage(UsageInput{Status: doneStatus(), Usage: usage})
	assert.Equal(t, NotApplicable, v.PhaseTimeline.Availability)
	assert.Equal(t, int64(11), *rowFor(t, v.PhaseTimeline, "DISCOVERY").Tokens.Total)
}

func TestBuildUsageUnreadableUsageLedgerLeavesEveryTokenNull(t *testing.T) {
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), UsageError: errors.New("x")})
	assert.Equal(t, Unavailable, v.PhaseTimeline.UsageAvailability)
	assert.Nil(t, v.PhaseTimeline.Total.Tokens.Input)
	require.Len(t, v.Diagnostics, 1)
	assert.Equal(t, ReasonMissionViewUsageUnavailable, v.Diagnostics[0].Reason)
}

func TestBuildUsageViewAndUsageShareTheTimelineWithUsage(t *testing.T) {
	usage := []telemetry.MissionUsageRecord{inc("a", "DISCOVERY", "inv_1", 1, n(10), n(1), "2026-10-10T10:10:00Z")}
	view := Build(Input{Status: doneStatus(), Registry: domain.DefaultRoleRegistry(), Lifecycle: fullPipeline(), Usage: usage})
	usageView := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), Usage: usage})
	a, err := json.Marshal(view.PhaseTimeline)
	require.NoError(t, err)
	b, err := json.Marshal(usageView.PhaseTimeline)
	require.NoError(t, err)
	assert.JSONEq(t, string(a), string(b))
}

func TestBuildChronologyMergesMarkersAndObservationsInTimeOrder(t *testing.T) {
	usage := []telemetry.MissionUsageRecord{
		snap("s1", "sess-1", "REFINEMENT", n(40000), n(1), "2026-10-10T10:30:00Z"),
	}
	got := BuildChronology(fullPipeline()[:9], usage)
	var kinds []string
	for _, e := range got {
		kinds = append(kinds, e.Kind)
	}
	idx := -1
	for i, k := range kinds {
		if k == "usage_observation" {
			idx = i
		}
	}
	require.GreaterOrEqual(t, idx, 0)
	assert.Equal(t, "phase_started", kinds[idx-1], "refinement started at 10:24, before the 10:30 snapshot")
	assert.Contains(t, got[idx].Detail, "cumulative_snapshot in=40000")
}

func TestRenderUsageHumanShowsTotalsCoverageAndChronology(t *testing.T) {
	usage := []telemetry.MissionUsageRecord{inc("a", "DISCOVERY", "inv_1", 1, n(1000), n(200), "2026-10-10T10:10:00Z")}
	v := BuildUsage(UsageInput{Status: doneStatus(), Lifecycle: fullPipeline(), Usage: usage, KnownInvocations: []string{"inv_1", "inv_9"}, WithChronology: true})
	var out bytes.Buffer
	require.NoError(t, RenderUsageHuman(&out, v, false))
	s := out.String()
	assert.Contains(t, s, "MISSION TOTAL   in=1000 out=200 total=1200   quality: reported_unverified   observations: 1")
	assert.Contains(t, s, "coverage: invocations known=2 observed=1 missing=1")
	assert.Contains(t, s, "Timeline")
	assert.Contains(t, s, "usage_observation")
	assert.NotContains(t, s, "MISSION TOTAL   tokens: —")
}

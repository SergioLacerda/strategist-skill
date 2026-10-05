package missionview

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lr(kind, phase, run, reason, at string) telemetry.MissionLifecycleRecord {
	return telemetry.MissionLifecycleRecord{
		SchemaVersion: telemetry.MissionLifecycleSchemaVersion,
		MarkerID:      kind + "|" + phase + "|" + run + "|" + reason + "|" + at,
		MarkerKind:    kind, MissionID: "m-1", Phase: phase, PhaseRunID: run,
		EndReason: reason, ObservedAt: at,
	}
}

func rowFor(t *testing.T, tl PhaseTimeline, phase string) PhaseRow {
	t.Helper()
	for _, r := range tl.Phases {
		if r.Phase == phase {
			return r
		}
	}
	require.Failf(t, "row not found", "phase %s", phase)
	return PhaseRow{}
}

func strVal(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func doneStatus() domain.MissionEngineStatus {
	return domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseDone, State: domain.StateDoneDelivery}
}

func fullPipeline() []telemetry.MissionLifecycleRecord {
	return []telemetry.MissionLifecycleRecord{
		lr("mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lr("phase_started", "BOOTSTRAP", "r1", "", "2026-10-10T10:00:00Z"),
		lr("phase_ended", "BOOTSTRAP", "r1", "completed", "2026-10-10T10:01:00Z"),
		lr("phase_started", "INTAKE", "r1", "", "2026-10-10T10:01:00Z"),
		lr("phase_ended", "INTAKE", "r1", "completed", "2026-10-10T10:02:00Z"),
		lr("phase_started", "DISCOVERY", "r1", "", "2026-10-10T10:02:00Z"),
		lr("phase_ended", "DISCOVERY", "r1", "completed", "2026-10-10T10:24:00Z"),
		lr("phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"),
		lr("phase_ended", "REFINEMENT", "r1", "completed", "2026-10-10T11:00:00Z"),
		lr("phase_started", "APPROVAL_GATE", "r1", "", "2026-10-10T11:00:00Z"),
		lr("phase_ended", "APPROVAL_GATE", "r1", "completed", "2026-10-10T11:05:00Z"),
		lr("phase_started", "EXECUTION", "r1", "", "2026-10-10T11:05:00Z"),
		lr("phase_ended", "EXECUTION", "r1", "completed", "2026-10-10T11:25:00Z"),
		lr("mission_ended", "DONE", "", "completed", "2026-10-10T11:25:00Z"),
	}
}

func TestBuildPhaseTimelineFullPipeline(t *testing.T) {
	tl := BuildPhaseTimeline(doneStatus(), fullPipeline(), nil)

	assert.Equal(t, Available, tl.Availability)
	assert.Equal(t, "completed", tl.Status)
	assert.Equal(t, "2026-10-10T10:00:00Z", strVal(tl.StartedAt))
	assert.Equal(t, "2026-10-10T11:25:00Z", strVal(tl.EndedAt))
	require.NotNil(t, tl.DurationSeconds)
	assert.Equal(t, int64(85*60), *tl.DurationSeconds)
	require.Len(t, tl.Phases, 5)

	intake := rowFor(t, tl, "INTAKE")
	assert.Equal(t, "Intake", intake.Label)
	assert.Equal(t, 2, intake.Runs, "bootstrap and intake form one row")
	assert.Equal(t, "2026-10-10T10:00:00Z", strVal(intake.StartedAt))
	assert.Equal(t, "2026-10-10T10:02:00Z", strVal(intake.EndedAt))

	discovery := rowFor(t, tl, "DISCOVERY")
	assert.Equal(t, "completed", discovery.Status)
	require.NotNil(t, discovery.DurationSeconds)
	assert.Equal(t, int64(22*60), *discovery.DurationSeconds)

	gate := rowFor(t, tl, "APPROVAL_GATE")
	assert.Equal(t, "Gate", gate.Label)
	assert.Equal(t, "not_applicable", gate.Quality)
	assert.Equal(t, "Delivery", rowFor(t, tl, "EXECUTION").Label)
}

func TestBuildPhaseTimelineNeverRendersUnobservedTokensAsZero(t *testing.T) {
	tl := BuildPhaseTimeline(doneStatus(), fullPipeline(), nil)
	raw, err := json.Marshal(tl)
	require.NoError(t, err)
	var generic struct {
		Phases []struct {
			Tokens  map[string]any `json:"tokens"`
			Quality string         `json:"quality"`
		} `json:"phases"`
	}
	require.NoError(t, json.Unmarshal(raw, &generic))
	for _, p := range generic.Phases {
		for key, v := range p.Tokens {
			assert.Nil(t, v, "token member %s must be null", key)
		}
		assert.Contains(t, []string{"unavailable", "not_applicable"}, p.Quality)
	}
}

func TestBuildPhaseTimelineShortRouteSkipsUnstartedPhases(t *testing.T) {
	recs := []telemetry.MissionLifecycleRecord{
		lr("mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lr("phase_started", "BOOTSTRAP", "r1", "", "2026-10-10T10:00:00Z"),
		lr("phase_ended", "BOOTSTRAP", "r1", "completed", "2026-10-10T10:01:00Z"),
		lr("phase_started", "INTAKE", "r1", "", "2026-10-10T10:01:00Z"),
		lr("phase_ended", "INTAKE", "r1", "completed", "2026-10-10T10:02:00Z"),
		lr("phase_started", "APPROVAL_GATE", "r1", "", "2026-10-10T10:02:00Z"),
		lr("phase_ended", "APPROVAL_GATE", "r1", "completed", "2026-10-10T10:03:00Z"),
		lr("phase_started", "EXECUTION", "r1", "", "2026-10-10T10:03:00Z"),
		lr("phase_ended", "EXECUTION", "r1", "completed", "2026-10-10T10:05:00Z"),
		lr("mission_ended", "DONE", "", "completed", "2026-10-10T10:05:00Z"),
	}
	tl := BuildPhaseTimeline(doneStatus(), recs, nil)
	for _, phase := range []string{"DISCOVERY", "REFINEMENT"} {
		row := rowFor(t, tl, phase)
		assert.Equal(t, "skipped", row.Status)
		assert.Nil(t, row.StartedAt)
		assert.Nil(t, row.DurationSeconds)
		assert.Zero(t, row.Runs)
	}
}

func TestBuildPhaseTimelineRunningMissionHasNoEndAndNeverInterrupted(t *testing.T) {
	recs := fullPipeline()[:8] // refinement started, not ended
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	tl := BuildPhaseTimeline(status, recs, nil)

	assert.Equal(t, "running", tl.Status)
	assert.Nil(t, tl.EndedAt)
	assert.Nil(t, tl.DurationSeconds)
	assert.Equal(t, "2026-10-10T10:24:00Z", strVal(tl.LastMarkerAt))
	refining := rowFor(t, tl, "REFINEMENT")
	assert.Equal(t, "running", refining.Status)
	assert.Nil(t, refining.EndedAt)
	assert.Nil(t, refining.DurationSeconds)
	assert.Equal(t, "not_started", rowFor(t, tl, "APPROVAL_GATE").Status)
	assert.NotEqual(t, "interrupted", tl.Status)
}

func TestBuildPhaseTimelineGateRevisionCountsTwoRefiningRuns(t *testing.T) {
	recs := []telemetry.MissionLifecycleRecord{
		lr("mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lr("phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"),
		lr("phase_ended", "REFINEMENT", "r1", "completed", "2026-10-10T11:00:00Z"),
		lr("phase_started", "APPROVAL_GATE", "r1", "", "2026-10-10T11:00:00Z"),
		lr("phase_ended", "APPROVAL_GATE", "r1", "revision_requested", "2026-10-10T11:05:00Z"),
		lr("phase_started", "REFINEMENT", "r2", "", "2026-10-10T11:06:00Z"),
		lr("phase_ended", "REFINEMENT", "r2", "completed", "2026-10-10T11:30:00Z"),
	}
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseApprovalGate, State: domain.StateApprovalGate}
	tl := BuildPhaseTimeline(status, recs, nil)

	refining := rowFor(t, tl, "REFINEMENT")
	assert.Equal(t, 2, refining.Runs)
	require.NotNil(t, refining.DurationSeconds)
	assert.Equal(t, int64(60*60), *refining.DurationSeconds, "36m + 24m")
	require.Len(t, refining.RunDetails, 2)
	assert.Equal(t, "r1", refining.RunDetails[0].RunID)
	assert.Equal(t, "r2", refining.RunDetails[1].RunID)
	assert.Equal(t, "gap", rowFor(t, tl, "APPROVAL_GATE").Status, "engine is in the gate but its second run was never recorded")
}

func TestBuildPhaseTimelineBlockedMissionKeepsLaterPhasesNotStarted(t *testing.T) {
	recs := []telemetry.MissionLifecycleRecord{
		lr("mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lr("phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"),
		lr("phase_ended", "REFINEMENT", "r1", "blocked", "2026-10-10T10:40:00Z"),
	}
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseBlocked, State: domain.StateBlocked}
	tl := BuildPhaseTimeline(status, recs, nil)

	assert.Equal(t, "blocked", tl.Status)
	assert.Nil(t, tl.EndedAt, "no mission_ended marker exists for a blocked mission")
	assert.Equal(t, "blocked", rowFor(t, tl, "REFINEMENT").Status)
	assert.Equal(t, "not_started", rowFor(t, tl, "EXECUTION").Status)
}

func TestBuildPhaseTimelineADRReentryKeepsTheMissionOutcome(t *testing.T) {
	recs := append(fullPipeline(),
		lr("phase_started", "APPROVAL_GATE", "r2", "", "2026-10-10T11:30:00Z"),
		lr("phase_ended", "APPROVAL_GATE", "r2", "completed", "2026-10-10T11:40:00Z"),
		lr("mission_ended", "DONE", "", "adr_closed", "2026-10-10T11:40:00Z"),
	)
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseDone, State: domain.StateADRDone}
	tl := BuildPhaseTimeline(status, recs, nil)

	assert.Equal(t, "completed", tl.Status, "the first mission end is the outcome")
	assert.Equal(t, "2026-10-10T11:40:00Z", strVal(tl.EndedAt), "the last mission end closes the timeline")
	assert.Equal(t, 2, rowFor(t, tl, "APPROVAL_GATE").Runs)
}

func TestBuildPhaseTimelineMissingLedgerIsNotRecordedNeverZero(t *testing.T) {
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	tl := BuildPhaseTimeline(status, nil, nil)

	assert.Equal(t, NotApplicable, tl.Availability)
	assert.Equal(t, "no lifecycle recorded", tl.Note)
	assert.Nil(t, tl.StartedAt)
	assert.Nil(t, tl.DurationSeconds)
	for _, row := range tl.Phases {
		assert.Equal(t, "not_recorded", row.Status)
		assert.Zero(t, row.Runs)
	}
}

func TestBuildPhaseTimelineReadErrorIsUnavailable(t *testing.T) {
	tl := BuildPhaseTimeline(doneStatus(), nil, errors.New("boom"))
	assert.Equal(t, Unavailable, tl.Availability)
	assert.Nil(t, tl.StartedAt)
}

func TestBuildPhaseTimelineEngineDoneWithoutEndMarkerIsAGap(t *testing.T) {
	recs := fullPipeline()[:13] // no mission_ended
	tl := BuildPhaseTimeline(doneStatus(), recs, nil)
	assert.Equal(t, "gap", tl.Status)
	assert.Nil(t, tl.EndedAt)
}

func TestBuildPhaseTimelineCarriesStage(t *testing.T) {
	recs := fullPipeline()
	recs[0].Stage = "SHORT"
	tl := BuildPhaseTimeline(doneStatus(), recs, nil)
	assert.Equal(t, "SHORT", tl.Route)
}

func TestBuildPhaseTimelineOpenRunTheEngineLeftIsAGap(t *testing.T) {
	recs := []telemetry.MissionLifecycleRecord{
		lr("mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lr("phase_started", "BOOTSTRAP", "r1", "", "2026-10-10T10:00:00Z"),
	}
	status := domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseIntake, State: domain.StateInit}
	tl := BuildPhaseTimeline(status, recs, nil)

	assert.Equal(t, "gap", rowFor(t, tl, "INTAKE").Status, "bootstrap end marker is missing while the engine is in intake")
	assert.Nil(t, rowFor(t, tl, "INTAKE").EndedAt)
}

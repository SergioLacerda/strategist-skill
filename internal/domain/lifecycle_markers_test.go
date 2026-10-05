package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type markerWant struct {
	kind   LifecycleMarkerKind
	phase  PipelinePhase
	reason LifecycleEndReason
}

func wantMarkers(t *testing.T, got []LifecycleMarker, want ...markerWant) {
	t.Helper()
	require.Len(t, got, len(want))
	for i, w := range want {
		require.Equal(t, w.kind, got[i].Kind, "marker %d kind", i)
		require.Equal(t, w.phase, got[i].Phase, "marker %d phase", i)
		require.Equal(t, w.reason, got[i].EndReason, "marker %d reason", i)
	}
}

func submitFrom(t *testing.T, status MissionEngineStatus, event MissionEngineEvent) (MissionEngineStatus, []LifecycleMarker) {
	t.Helper()
	engine, err := RestoreMission(status)
	require.NoError(t, err)
	after, err := engine.Submit(event)
	require.NoError(t, err)
	return after, LifecycleMarkersForTransition(event, status, after)
}

func TestLifecycleMarkersForStart(t *testing.T) {
	_, status, err := StartMission(MissionStartRequest{MissionID: "m-1"})
	require.NoError(t, err)
	wantMarkers(t, LifecycleMarkersForStart(status),
		markerWant{kind: MarkerMissionStarted, phase: PhaseBootstrap},
		markerWant{kind: MarkerPhaseStarted, phase: PhaseBootstrap},
	)
}

func TestLifecycleMarkersFullPipelineSequence(t *testing.T) {
	status := MissionEngineStatus{MissionID: "m-1", Phase: PhaseBootstrap, State: StateInit}
	steps := []struct {
		event MissionEngineEvent
		want  []markerWant
	}{
		{MissionEventBootstrapDone, []markerWant{{MarkerPhaseEnded, PhaseBootstrap, EndCompleted}, {MarkerPhaseStarted, PhaseIntake, ""}}},
		{MissionEventIntakeDone, []markerWant{{MarkerPhaseEnded, PhaseIntake, EndCompleted}, {MarkerPhaseStarted, PhaseDiscovery, ""}}},
		{MissionEventDiscoveryDone, []markerWant{{MarkerPhaseEnded, PhaseDiscovery, EndCompleted}, {MarkerPhaseStarted, PhaseRefinement, ""}}},
		{MissionEventRefinementDone, []markerWant{{MarkerPhaseEnded, PhaseRefinement, EndCompleted}, {MarkerPhaseStarted, PhaseApprovalGate, ""}}},
		// the handoff challenge stays inside the Approval Gate phase: no marker
		{MissionEventGateApproved, nil},
		{MissionEventHandoffSatisfied, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndCompleted}, {MarkerPhaseStarted, PhaseExecution, ""}}},
		{MissionEventSniperDone, []markerWant{{MarkerPhaseEnded, PhaseExecution, EndCompleted}, {MarkerMissionEnded, PhaseDone, EndCompleted}}},
	}
	for _, step := range steps {
		var got []LifecycleMarker
		status, got = submitFrom(t, status, step.event)
		wantMarkers(t, got, step.want...)
	}
}

func TestLifecycleMarkersFromGateAndFailureEvents(t *testing.T) {
	gate := MissionEngineStatus{MissionID: "m-1", Phase: PhaseApprovalGate, State: StateApprovalGate}
	handoff := MissionEngineStatus{MissionID: "m-1", Phase: PhaseApprovalGate, State: StateHandoffChallenge}
	refinement := MissionEngineStatus{MissionID: "m-1", Phase: PhaseRefinement, State: StateRefinement}
	execution := MissionEngineStatus{MissionID: "m-1", Phase: PhaseExecution, State: StateExecution}

	cases := []struct {
		name   string
		from   MissionEngineStatus
		event  MissionEngineEvent
		expect []markerWant
	}{
		{"revision re-enters refinement", gate, MissionEventGateRevision, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndRevisionRequested}, {MarkerPhaseStarted, PhaseRefinement, ""}}},
		{"gate denied ends the mission rejected", gate, MissionEventGateDenied, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndRejected}, {MarkerMissionEnded, PhaseDone, EndRejected}}},
		{"gate timeout", gate, MissionEventGateTimeout, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndTimedOut}, {MarkerMissionEnded, PhaseDone, EndTimedOut}}},
		{"analysis only", gate, MissionEventGateApprovedAnalysisOnly, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndAnalysisDelivered}, {MarkerMissionEnded, PhaseDone, EndAnalysisDelivered}}},
		{"handoff failed returns to refinement", handoff, MissionEventHandoffFailed, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndRevisionRequested}, {MarkerPhaseStarted, PhaseRefinement, ""}}},
		{"handoff exhausted blocks", handoff, MissionEventHandoffExhausted, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndBlocked}}},
		{"handoff not applicable", handoff, MissionEventHandoffNotApplicable, []markerWant{{MarkerPhaseEnded, PhaseApprovalGate, EndAnalysisDelivered}, {MarkerMissionEnded, PhaseDone, EndAnalysisDelivered}}},
		{"no tasks", refinement, MissionEventNoTasks, []markerWant{{MarkerPhaseEnded, PhaseRefinement, EndAnalysisDelivered}, {MarkerMissionEnded, PhaseDone, EndAnalysisDelivered}}},
		{"permanent failure in refinement", refinement, MissionEventSlotPermanent, []markerWant{{MarkerPhaseEnded, PhaseRefinement, EndBlocked}}},
		{"permanent failure in execution", execution, MissionEventSlotPermanent, []markerWant{{MarkerPhaseEnded, PhaseExecution, EndBlocked}}},
		{"invalid artifact re-enters refinement", execution, MissionEventRefinementArtifactInvalid, []markerWant{{MarkerPhaseEnded, PhaseExecution, EndRevisionRequested}, {MarkerPhaseStarted, PhaseRefinement, ""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := submitFrom(t, tc.from, tc.event)
			wantMarkers(t, got, tc.expect...)
		})
	}
}

func TestLifecycleMarkersRetriesStayInsideThePhase(t *testing.T) {
	refinement := MissionEngineStatus{MissionID: "m-1", Phase: PhaseRefinement, State: StateRefinement}
	retrying, got := submitFrom(t, refinement, MissionEventSlotTransient)
	require.Equal(t, StateRetryingRefinement, retrying.State)
	require.Empty(t, got)

	back, got := submitFrom(t, retrying, MissionEventRetryOK)
	require.Equal(t, StateRefinement, back.State)
	require.Empty(t, got)

	_, got = submitFrom(t, retrying, MissionEventSlotTransient)
	wantMarkers(t, got, markerWant{MarkerPhaseEnded, PhaseRefinement, EndBlocked})
}

func TestLifecycleMarkersCriticalHitJumpLeavesPhasesUnstarted(t *testing.T) {
	status := MissionEngineStatus{
		MissionID: "m-1", Phase: PhaseIntake, State: StateInit,
		Stage: StageShort, StageFeat: "critical_hit", StageGateRequired: true, StageCorrelationID: "corr-1",
	}
	after, got := submitFrom(t, status, MissionEventCriticalHitIntent)
	require.Equal(t, PhaseApprovalGate, after.Phase)
	wantMarkers(t, got,
		markerWant{MarkerPhaseEnded, PhaseIntake, EndCompleted},
		markerWant{MarkerPhaseStarted, PhaseApprovalGate, ""},
	)
	require.Equal(t, StageShort, got[0].Stage)
}

func TestLifecycleMarkersADRReentryAfterDone(t *testing.T) {
	done := MissionEngineStatus{MissionID: "m-1", Phase: PhaseDone, State: StateDoneDelivery}
	gate1, got := submitFrom(t, done, MissionEventADRCriterion)
	require.Equal(t, PhaseApprovalGate, gate1.Phase)
	wantMarkers(t, got, markerWant{kind: MarkerPhaseStarted, phase: PhaseApprovalGate})

	gate2, got := submitFrom(t, gate1, MissionEventADRApproved)
	require.Equal(t, StateADRGate2, gate2.State)
	require.Empty(t, got)

	_, got = submitFrom(t, gate2, MissionEventADRDeclined)
	wantMarkers(t, got,
		markerWant{MarkerPhaseEnded, PhaseApprovalGate, EndCompleted},
		markerWant{MarkerMissionEnded, PhaseDone, EndADRClosed},
	)
}

func TestLifecycleMarkersSideQuestExecutionReentersRefinement(t *testing.T) {
	before := MissionEngineStatus{MissionID: "m-1", Phase: PhaseExecution, State: StateSideQuestExec}
	after := MissionEngineStatus{MissionID: "m-1", Phase: PhaseRefinement, State: StateRefinement}
	got := LifecycleMarkersForTransition(MissionEventSniperDone, before, after)
	wantMarkers(t, got,
		markerWant{MarkerPhaseEnded, PhaseExecution, EndCompleted},
		markerWant{MarkerPhaseStarted, PhaseRefinement, ""},
	)
}

func TestLifecycleMarkersDirectDoneAndSamePhase(t *testing.T) {
	before := MissionEngineStatus{MissionID: "m-1", Phase: PhaseExecution, State: StateDirectExec}
	after := MissionEngineStatus{MissionID: "m-1", Phase: PhaseDone, State: StateDirectDone}
	wantMarkers(t, LifecycleMarkersForTransition(MissionEventSniperDone, before, after),
		markerWant{MarkerPhaseEnded, PhaseExecution, EndCompleted},
		markerWant{MarkerMissionEnded, PhaseDone, EndCompleted},
	)
	require.Empty(t, LifecycleMarkersForTransition(MissionEventSlotTransient, before, before))
}

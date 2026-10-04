package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMissionEngine_RejectsOutOfOrderEventsAndPreservesState(t *testing.T) {
	engine, status, err := StartMission(MissionStartRequest{MissionID: "m-1"})
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != PhaseBootstrap || status.State != StateInit {
		t.Fatalf("initial status = %+v", status)
	}
	if _, err := engine.Submit(MissionEventDiscoveryDone); err == nil {
		t.Fatal("expected out-of-order event to fail")
	}
	if got := engine.Status(); got != status {
		t.Fatalf("status changed after rejected event: got %+v want %+v", got, status)
	}
}

func TestMissionEngine_FullPipelineProgression(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-2"})
	if err != nil {
		t.Fatal(err)
	}
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone,
		MissionEventIntakeDone,
		MissionEventDiscoveryDone,
		MissionEventRefinementDone,
		MissionEventGateApproved,
	})
	if got := engine.Status(); got.State != StateHandoffChallenge {
		t.Fatalf("state before handoff = %q", got.State)
	}
	if _, err := engine.RecordHandoffEvaluation(HandoffEvaluation{Attempt: 1, MaxAttempts: 2, Result: HandoffEvaluationPassed, Status: "passed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Submit(MissionEventHandoffSatisfied); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Submit(MissionEventSniperDone); err != nil {
		t.Fatal(err)
	}
	if got := engine.Status(); got.Phase != PhaseDone || got.State != StateDoneDelivery {
		t.Fatalf("final status = %+v", got)
	}
}

func TestMissionEngine_CriticalHitStageProgressionUsesOrdinaryExecution(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-critical-hit"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{MissionEventBootstrapDone})
	_, err = engine.RecordStageResolution(StageResolution{
		SchemaVersion: StageResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion,
		Stage: StageShort, LegacyRoute: "critical_hit", Feat: "critical_hit", MissionID: "m-critical-hit",
		CorrelationKey: "m-critical-hit:critical_hit", GateRequired: true,
		PolicyVersion: "route-resolution/v1", Reason: "Critical Hit compatibility route bounded to SHORT",
	})
	require.NoError(t, err)
	status, err := engine.Submit(MissionEventCriticalHitIntent)
	require.NoError(t, err)
	require.Equal(t, PhaseApprovalGate, status.Phase)
	require.Equal(t, StateApprovalGate, status.State)
	require.False(t, status.StageGateApproved)
	_, err = engine.Submit(MissionEventGateApproved)
	require.ErrorContains(t, err, "explicit Critical Hit gate event")

	status, err = engine.Submit(MissionEventCriticalHitGateApproved)
	require.NoError(t, err)
	require.Equal(t, PhaseExecution, status.Phase)
	require.Equal(t, StateExecution, status.State)
	require.True(t, status.StageGateApproved)
	require.Equal(t, CriticalHitStageApprovalDigest(status), status.ApprovalGatePackageDigest)

	status, err = engine.Submit(MissionEventSniperDone)
	require.NoError(t, err)
	require.Equal(t, StateDoneDelivery, status.State)
}

func TestMissionEngine_CriticalHitIntentRequiresRecordedResolution(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-critical-hit-missing-resolution"})
	require.NoError(t, err)
	_, err = engine.Submit(MissionEventBootstrapDone)
	require.NoError(t, err)
	before := engine.Status()
	_, err = engine.Submit(MissionEventCriticalHitIntent)
	require.ErrorContains(t, err, "recorded gated SHORT resolution")
	require.Equal(t, before, engine.Status())
}

func TestMissionEngine_CriticalHitDeclineNeverEntersExecution(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-critical-hit-decline"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{MissionEventBootstrapDone})
	_, err = engine.RecordStageResolution(StageResolution{
		SchemaVersion: StageResolutionArtifactSchemaVersion, TaxonomyVersion: CanonicalTaxonomyVersion,
		Stage: StageShort, LegacyRoute: "critical_hit", Feat: "critical_hit", MissionID: "m-critical-hit-decline",
		CorrelationKey: "m-critical-hit-decline:critical_hit", GateRequired: true,
		PolicyVersion: "route-resolution/v1", Reason: "Critical Hit compatibility route bounded to SHORT",
	})
	require.NoError(t, err)
	_, err = engine.Submit(MissionEventCriticalHitIntent)
	require.NoError(t, err)
	status, err := engine.Submit(MissionEventCriticalHitGateDeclined)
	require.NoError(t, err)
	require.Equal(t, PhaseDone, status.Phase)
	require.Equal(t, StateDoneAnalysis, status.State)
	require.False(t, status.StageGateApproved)
	_, err = engine.Submit(MissionEventSniperDone)
	require.Error(t, err)
}

func TestMissionEngine_InvalidRefinementArtifactReturnsToRefinement(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-artifact-repair"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone,
		MissionEventIntakeDone,
		MissionEventDiscoveryDone,
		MissionEventRefinementDone,
		MissionEventGateApproved,
		MissionEventHandoffSatisfied,
	})

	status, err := engine.Submit(MissionEventRefinementArtifactInvalid)
	require.NoError(t, err)
	require.Equal(t, StateRefinement, status.State)
	require.Equal(t, PhaseRefinement, status.Phase)
}

func TestMissionEngine_InvalidRefinementArtifactDoesNotReplacePermanentFailure(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-artifact-failure"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone,
		MissionEventIntakeDone,
		MissionEventDiscoveryDone,
		MissionEventRefinementDone,
		MissionEventGateApproved,
		MissionEventHandoffSatisfied,
	})

	status, err := engine.Submit(MissionEventSlotPermanent)
	require.NoError(t, err)
	require.Equal(t, StateBlocked, status.State)

	_, err = engine.Submit(MissionEventRefinementArtifactInvalid)
	require.Error(t, err)
	require.Equal(t, StateBlocked, engine.Status().State)
}

func TestMissionEngine_ApprovalGatePackageDigestRequiresExplicitBinding(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-gate-digest"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone,
		MissionEventIntakeDone,
		MissionEventDiscoveryDone,
		MissionEventRefinementDone,
	})

	_, err = engine.RecordApprovalGatePackageDigest("sha256:before-gate")
	require.Error(t, err)

	submitMissionEvents(t, engine, []MissionEngineEvent{MissionEventGateApproved})
	status, err := engine.RecordApprovalGatePackageDigest("sha256:approved")
	require.NoError(t, err)
	require.Equal(t, "sha256:approved", status.ApprovalGatePackageDigest)

	status, err = engine.Submit(MissionEventHandoffFailed)
	require.NoError(t, err)
	require.Empty(t, status.ApprovalGatePackageDigest)
}

func TestMissionEngine_NewGateDoesNotCarryAnOldPackageDigest(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-gate-refresh"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone,
		MissionEventIntakeDone,
		MissionEventDiscoveryDone,
		MissionEventRefinementDone,
		MissionEventGateApproved,
	})
	_, err = engine.RecordApprovalGatePackageDigest("sha256:first")
	require.NoError(t, err)

	_, err = engine.RecordHandoffEvaluation(HandoffEvaluation{
		Attempt: 1, MaxAttempts: 2, Result: HandoffEvaluationFailed, Status: "failed",
	})
	require.NoError(t, err)
	require.Empty(t, engine.Status().ApprovalGatePackageDigest)

	submitMissionEvents(t, engine, []MissionEngineEvent{MissionEventRefinementDone, MissionEventGateApproved})
	require.Empty(t, engine.Status().ApprovalGatePackageDigest)
}

func submitMissionEvents(t *testing.T, engine *MissionEngine, events []MissionEngineEvent) {
	t.Helper()
	for _, event := range events {
		if _, err := engine.Submit(event); err != nil {
			t.Fatalf("submit %q: %v", event, err)
		}
	}
}

func TestMissionEngine_RestoresAndRejectsInvalidSnapshots(t *testing.T) {
	status := MissionEngineStatus{MissionID: "m-3", Phase: PhaseExecution, State: StateExecution}
	engine, err := RestoreMission(status)
	if err != nil {
		t.Fatal(err)
	}
	if got := engine.Status(); got != status {
		t.Fatalf("restored status = %+v want %+v", got, status)
	}
	if _, err := RestoreMission(MissionEngineStatus{MissionID: "m-4", Phase: PhaseBlocked, State: StateExecution}); err == nil {
		t.Fatal("expected inconsistent blocked snapshot to fail")
	}
	if _, err := RestoreMission(MissionEngineStatus{MissionID: "m-5", Phase: PhaseExecution, State: StateApprovalGate}); err == nil {
		t.Fatal("expected cross-phase persisted state to fail")
	}
}

func TestMissionEngine_RestoredEarlyPhaseAdvancesThroughFacade(t *testing.T) {
	engine, err := RestoreMission(MissionEngineStatus{
		MissionID: "m-restore-early",
		Phase:     PhaseDiscovery,
		State:     StateInit,
	})
	require.NoError(t, err)

	status, err := engine.Submit(MissionEventDiscoveryDone)
	require.NoError(t, err)
	require.Equal(t, PhaseRefinement, status.Phase)
	require.Equal(t, StateRefinement, status.State)

	_, err = engine.Submit(MissionEventDiscoveryDone)
	require.Error(t, err)
	require.Equal(t, status, engine.Status())
}

func TestMissionEngine_HandlesRetryRevisionAndPermanentBlock(t *testing.T) {
	engine := missionEngineAtGateAfterRevision(t)
	failHandoff(t, engine, 1)
	require.Equal(t, StateRefinement, engine.Status().State)
	require.Equal(t, 1, engine.Status().HandoffAttempt)
	require.NoError(t, submitEvent(engine, MissionEventRefinementDone))
	require.NoError(t, submitGateApproved(engine))
	failHandoff(t, engine, 2)
	require.Equal(t, StateBlocked, engine.Status().State)
	require.Error(t, submitRetry(engine))
}

func missionEngineAtGateAfterRevision(t *testing.T) *MissionEngine {
	t.Helper()
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-6"})
	require.NoError(t, err)
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone, MissionEventIntakeDone, MissionEventDiscoveryDone,
		MissionEventRefinementDone, MissionEventGateRevision, MissionEventRefinementDone,
	})
	require.NoError(t, submitGateApproved(engine))
	return engine
}

func submitGateApproved(engine *MissionEngine) error {
	_, err := engine.Submit(MissionEventGateApproved)
	return err
}

func failHandoff(t *testing.T, engine *MissionEngine, attempt int) {
	t.Helper()
	_, err := engine.RecordHandoffEvaluation(HandoffEvaluation{Attempt: attempt, MaxAttempts: 2, Result: HandoffEvaluationFailed, Status: "failed", NextAction: "return_to_archivist"})
	require.NoError(t, err)
}

func submitRetry(engine *MissionEngine) error {
	_, err := engine.Submit(MissionEventRetryOK)
	return err
}

func submitEvent(engine *MissionEngine, event MissionEngineEvent) error {
	_, err := engine.Submit(event)
	return err
}

func TestMissionEngine_HandoffCannotBypassApprovalGate(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-8"})
	if err != nil {
		t.Fatal(err)
	}
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone, MissionEventIntakeDone, MissionEventDiscoveryDone,
		MissionEventRefinementDone,
	})
	if got := engine.Status(); got.State != StateApprovalGate {
		t.Fatalf("state before approval = %q", got.State)
	}
	if _, err := engine.RecordHandoffEvaluation(HandoffEvaluation{Attempt: 1, MaxAttempts: 2, Result: HandoffEvaluationPassed, Status: "passed"}); err == nil {
		t.Fatal("expected handoff evaluation to require the independent Approval Gate")
	}
}

func TestReplayMissionRejectsGapsAndUsesMissionTransitions(t *testing.T) {
	snapshot := MissionEngineStatus{MissionID: "replay", Phase: PhaseBootstrap, State: StateInit}
	engine, err := ReplayMission(snapshot, []MissionReplayEvent{{Sequence: 1, Event: MissionEventBootstrapDone}})
	require.NoError(t, err)
	require.Equal(t, PhaseIntake, engine.Status().Phase)
	_, err = ReplayMission(snapshot, []MissionReplayEvent{{Sequence: 2, Event: MissionEventBootstrapDone}})
	require.ErrorContains(t, err, "sequence gap")
}

func TestRestoreMissionRejectsInvalidHandoffMetadata(t *testing.T) {
	_, err := RestoreMission(MissionEngineStatus{MissionID: "bad", Phase: PhaseExecution, State: StateExecution, HandoffStatus: "failed"})
	require.ErrorContains(t, err, "handoff metadata")
}

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
	if _, err := engine.SubmitHandoff(HandoffOutcome{Attempt: 1, MaxAttempts: 2, Passed: true, Status: "passed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Submit(MissionEventSniperDone); err != nil {
		t.Fatal(err)
	}
	if got := engine.Status(); got.Phase != PhaseDone || got.State != StateDoneDelivery {
		t.Fatalf("final status = %+v", got)
	}
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
	_, err := engine.SubmitHandoff(HandoffOutcome{Attempt: attempt, MaxAttempts: 2, Passed: false, Status: "failed", NextAction: "return_to_archivist"})
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

func TestMissionEngine_RejectsHandoffReplayAndInvalidAttempt(t *testing.T) {
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-7"})
	if err != nil {
		t.Fatal(err)
	}
	submitMissionEvents(t, engine, []MissionEngineEvent{
		MissionEventBootstrapDone, MissionEventIntakeDone, MissionEventDiscoveryDone,
		MissionEventRefinementDone, MissionEventGateApproved,
	})
	if _, err := engine.SubmitHandoff(HandoffOutcome{Attempt: 2, MaxAttempts: 2, Passed: true, Status: "passed"}); err == nil {
		t.Fatal("expected non-sequential attempt to fail")
	}
	if _, err := engine.SubmitHandoff(HandoffOutcome{Attempt: 1, MaxAttempts: 2, Passed: true, Status: "passed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.SubmitHandoff(HandoffOutcome{Attempt: 1, MaxAttempts: 2, Passed: true, Status: "passed"}); err == nil {
		t.Fatal("expected replay to fail after transition")
	}
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
	if _, err := engine.SubmitHandoff(HandoffOutcome{Attempt: 1, MaxAttempts: 2, Passed: true, Status: "passed"}); err == nil {
		t.Fatal("expected handoff to require the independent Approval Gate")
	}
}

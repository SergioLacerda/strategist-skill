package domain

import "testing"

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
		MissionEventSniperDone,
	})
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
	engine, _, err := StartMission(MissionStartRequest{MissionID: "m-6"})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []MissionEngineEvent{MissionEventBootstrapDone, MissionEventIntakeDone, MissionEventDiscoveryDone, MissionEventRefinementDone, MissionEventGateRevision, MissionEventRefinementDone, MissionEventGateApproved, MissionEventSlotTransient, MissionEventRetryOK, MissionEventSlotPermanent} {
		if _, err := engine.Submit(event); err != nil {
			t.Fatalf("submit %q: %v", event, err)
		}
	}
	if got := engine.Status(); got.Phase != PhaseBlocked || got.State != StateBlocked {
		t.Fatalf("blocked status = %+v", got)
	}
	if _, err := engine.Submit(MissionEventRetryOK); err == nil {
		t.Fatal("expected blocked mission to reject retry")
	}
}

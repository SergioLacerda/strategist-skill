package domain

import "fmt"

// MissionStartRequest identifies a mission at the transition boundary.
type MissionStartRequest struct {
	MissionID string
}

// MissionEngineStatus is the durable, implementation-neutral state needed to
// replay or restore a mission transition sequence.
type MissionEngineStatus struct {
	MissionID         string
	Phase             PipelinePhase
	State             MissionState
	HandoffAttempt    int
	HandoffStatus     string
	HandoffNextAction string
}

// HandoffOutcome is the already-verified and persisted result of the live
// Archivist-to-Sniper boundary. The domain consumes this neutral contract and
// does not depend on the handoff or telemetry packages.
type HandoffOutcome struct {
	Attempt     int
	MaxAttempts int
	Passed      bool
	NextAction  string
	Status      string
}

// MissionEngine is the mission-level facade over the two existing transition
// authorities. It deliberately does not expose a public CLI surface yet.
type MissionEngine struct {
	status MissionEngineStatus
	early  *PhaseTransitionAuthority
}

// StartMission creates a mission at the bootstrap phase and INIT state.
func StartMission(request MissionStartRequest) (*MissionEngine, MissionEngineStatus, error) {
	if request.MissionID == "" {
		return nil, MissionEngineStatus{}, fmt.Errorf("mission engine: mission id is required")
	}
	engine := &MissionEngine{
		status: MissionEngineStatus{
			MissionID: request.MissionID,
			Phase:     PhaseBootstrap,
			State:     StateInit,
		},
		early: NewPhaseTransitionAuthority(),
	}
	return engine, engine.status, nil
}

// RestoreMission validates and restores a previously persisted status.
func RestoreMission(status MissionEngineStatus) (*MissionEngine, error) {
	if err := validateMissionStatus(status); err != nil {
		return nil, err
	}
	engine := &MissionEngine{status: status, early: NewPhaseTransitionAuthority()}
	for _, event := range earlyEventsToPhase(status.Phase) {
		if _, err := engine.early.Submit(event); err != nil {
			return nil, fmt.Errorf("mission engine: restore phase %q: %w", status.Phase, err)
		}
	}
	return engine, nil
}

// Status returns the current durable status.
func (e *MissionEngine) Status() MissionEngineStatus {
	return e.status
}

// Submit applies one event. Invalid or out-of-order events leave the engine
// unchanged and return an error.
func (e *MissionEngine) Submit(event MissionEngineEvent) (MissionEngineStatus, error) {
	if e == nil || e.early == nil {
		return MissionEngineStatus{}, fmt.Errorf("mission engine: engine is nil")
	}
	if err := e.submitEarly(event); err == nil {
		return e.status, nil
	} else if e.status.Phase == PhaseBootstrap || e.status.Phase == PhaseIntake || e.status.Phase == PhaseDiscovery {
		return e.status, err
	}
	return e.submitFSM(event)
}

func (e *MissionEngine) submitEarly(event MissionEngineEvent) error {
	var phaseEvent PhaseEvent
	switch event {
	case MissionEventBootstrapDone:
		phaseEvent = EventBootstrapDone
	case MissionEventIntakeDone:
		phaseEvent = EventIntakeDone
	case MissionEventDiscoveryDone:
		phaseEvent = EventDiscoveryDone
	case MissionEventRefinementDone, MissionEventNoTasks,
		MissionEventGateApproved, MissionEventGateDenied, MissionEventGateTimeout,
		MissionEventGateRevision, MissionEventHandoffPassed, MissionEventHandoffFailed,
		MissionEventHandoffExhausted, MissionEventSniperDone, MissionEventRetryOK,
		MissionEventSlotTransient, MissionEventSlotPermanent, MissionEventADRCriterion,
		MissionEventADRApproved, MissionEventADRDeclined:
		return fmt.Errorf("mission engine: event %q is not an early-pipeline event", event)
	}
	next, err := e.early.Submit(phaseEvent)
	if err != nil {
		return err
	}
	e.status.Phase = next
	if next == PhaseRefinement {
		e.status.State = StateRefinement
	}
	return nil
}

// SubmitHandoff consumes one persisted live challenge result. It is the only
// route from the independent Approval Gate into Sniper execution. Failed
// attempts return to Archivist until the policy limit is reached, then remain
// blocked. Replaying an attempt is rejected because the state is no longer at
// the handoff boundary or the attempt number is not the next one.
func (e *MissionEngine) SubmitHandoff(outcome HandoffOutcome) (MissionEngineStatus, error) {
	if err := e.validateHandoffSubmission(outcome); err != nil {
		if e == nil {
			return MissionEngineStatus{}, err
		}
		return e.status, err
	}
	e.applyHandoffOutcome(outcome)
	return e.submitFSM(handoffOutcomeEvent(outcome))
}

func (e *MissionEngine) validateHandoffSubmission(outcome HandoffOutcome) error {
	if e == nil || e.early == nil {
		return fmt.Errorf("mission engine: engine is nil")
	}
	if e.status.State != StateHandoffChallenge {
		return fmt.Errorf("mission engine: handoff challenge is not pending from state %q", e.status.State)
	}
	if outcome.Attempt <= 0 || outcome.MaxAttempts <= 0 {
		return fmt.Errorf("mission engine: handoff attempt and max attempts must be positive")
	}
	if outcome.Attempt != e.status.HandoffAttempt+1 {
		return fmt.Errorf("mission engine: handoff attempt %d is not next attempt %d", outcome.Attempt, e.status.HandoffAttempt+1)
	}
	return nil
}

func (e *MissionEngine) applyHandoffOutcome(outcome HandoffOutcome) {
	e.status.HandoffAttempt = outcome.Attempt
	e.status.HandoffStatus = outcome.Status
	e.status.HandoffNextAction = outcome.NextAction
}

func handoffOutcomeEvent(outcome HandoffOutcome) MissionEngineEvent {
	if outcome.Passed {
		return MissionEventHandoffPassed
	}
	if outcome.Attempt >= outcome.MaxAttempts {
		return MissionEventHandoffExhausted
	}
	return MissionEventHandoffFailed
}

func (e *MissionEngine) submitFSM(event MissionEngineEvent) (MissionEngineStatus, error) {
	transition, ok := missionTransitionEvent(event)
	if !ok {
		return e.status, fmt.Errorf("mission engine: event %q is not valid from phase %q", event, e.status.Phase)
	}
	next := NextState(e.status.State, transition)
	if next == e.status.State {
		return e.status, fmt.Errorf("mission engine: event %q is not valid from state %q", event, e.status.State)
	}
	e.status.State = next
	e.status.Phase = phaseForState(next)
	return e.status, nil
}

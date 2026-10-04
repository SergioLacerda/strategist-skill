package domain

import "fmt"

// MissionStartRequest identifies a mission at the transition boundary.
type MissionStartRequest struct {
	MissionID string `json:"mission_id"`
}

// MissionEngineStatus is the durable, implementation-neutral state needed to
// replay or restore a mission transition sequence.
type MissionEngineStatus struct {
	MissionID string        `json:"mission_id"`
	Phase     PipelinePhase `json:"phase"`
	State     MissionState  `json:"state"`
	// Stage fields bind a short-stage request to the mission FSM. They are
	// optional for ordinary full-pipeline missions and required for Critical Hit.
	Stage              Stage  `json:"stage,omitempty"`
	StageFeat          string `json:"stage_feat,omitempty"`
	StageCorrelationID string `json:"stage_correlation_id,omitempty"`
	StageGateRequired  bool   `json:"stage_gate_required,omitempty"`
	StageGateApproved  bool   `json:"stage_gate_approved,omitempty"`
	// ApprovalGatePackageDigest binds the human gate acceptance to the exact
	// refined package that was reviewed. For Critical Hit SHORT it carries the
	// deterministic Stage approval digest instead of a refined-package digest.
	ApprovalGatePackageDigest string `json:"approval_gate_package_digest,omitempty"`
	HandoffAttempt            int    `json:"handoff_attempt,omitempty"`
	HandoffStatus             string `json:"handoff_status,omitempty"`
	HandoffNextAction         string `json:"handoff_next_action,omitempty"`
}

// MissionEngine is the single mission-level transition facade. The phase and
// state transition tables remain implementation details; callers cannot
// advance either authority independently.
type MissionEngine struct {
	status MissionEngineStatus
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
	}
	return engine, engine.status, nil
}

// RestoreMission validates and restores a previously persisted status.
func RestoreMission(status MissionEngineStatus) (*MissionEngine, error) {
	if err := validateMissionStatus(status); err != nil {
		return nil, err
	}
	return &MissionEngine{status: status}, nil
}

// Status returns the current durable status.
func (e *MissionEngine) Status() MissionEngineStatus {
	return e.status
}

// RecordApprovalGatePackageDigest binds the accepted main Approval Gate to the
// package revision that was reviewed. The command boundary computes the digest
// from the refined package while holding the mission lock; this method only
// commits that already-derived fact to the mission state.
func (e *MissionEngine) RecordApprovalGatePackageDigest(digest string) (MissionEngineStatus, error) {
	if e == nil {
		return MissionEngineStatus{}, fmt.Errorf("mission engine: engine is nil")
	}
	if e.status.State != StateHandoffChallenge {
		return e.status, fmt.Errorf("mission engine: approval gate package digest requires the handoff challenge state, got %q", e.status.State)
	}
	if digest == "" {
		return e.status, fmt.Errorf("mission engine: approval gate package digest is required")
	}
	e.status.ApprovalGatePackageDigest = digest
	return e.status, nil
}

// Submit applies one event. Invalid or out-of-order events leave the engine
// unchanged and return an error.
func (e *MissionEngine) Submit(event MissionEngineEvent) (MissionEngineStatus, error) {
	if e == nil {
		return MissionEngineStatus{}, fmt.Errorf("mission engine: engine is nil")
	}
	if err := rejectObsoleteEvent(event); err != nil {
		return e.status, err
	}
	if earlyMissionPhase(e.status.Phase) {
		if event == MissionEventCriticalHitIntent {
			return e.submitCriticalHitIntent()
		}
		return e.status, e.submitEarly(event)
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
		MissionEventGateApproved, MissionEventGateApprovedAnalysisOnly, MissionEventGateDenied, MissionEventGateTimeout,
		MissionEventGateRevision, MissionEventHandoffSatisfied, MissionEventHandoffFailed,
		MissionEventHandoffExhausted, MissionEventHandoffNotApplicable, MissionEventSniperDone, MissionEventRetryOK,
		MissionEventSlotTransient, MissionEventSlotPermanent, MissionEventRefinementArtifactInvalid, MissionEventADRCriterion,
		MissionEventADRApproved, MissionEventADRDeclined, MissionEventCriticalHitIntent,
		MissionEventCriticalHitGateApproved, MissionEventCriticalHitGateDeclined, obsoleteMissionEventHandoffPassed:
		return fmt.Errorf("mission engine: event %q is not an early-pipeline event", event)
	}
	transitions, ok := phaseTransitions[e.status.Phase]
	if !ok {
		return ErrOutOfOrderPhaseSubmit{Current: e.status.Phase, Event: phaseEvent}
	}
	next, ok := transitions[phaseEvent]
	if !ok {
		return ErrOutOfOrderPhaseSubmit{Current: e.status.Phase, Event: phaseEvent}
	}
	e.status.Phase = next
	if next == PhaseRefinement {
		e.status.State = StateRefinement
	}
	return nil
}

func (e *MissionEngine) submitFSM(event MissionEngineEvent) (MissionEngineStatus, error) {
	if err := e.validateCriticalHitGateEvent(event); err != nil {
		return e.status, err
	}
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
	e.recordCriticalHitGateOutcome(event)
	if event == MissionEventRefinementArtifactInvalid {
		e.status.HandoffStatus = ""
		e.status.HandoffNextAction = "reapprove_gate"
	}
	if next != StateExecution {
		// A new handoff challenge must be explicitly bound by the command
		// boundary after the gate event is accepted. Never carry a prior
		// package binding across a new gate or another state transition.
		e.status.ApprovalGatePackageDigest = ""
	}
	return e.status, nil
}

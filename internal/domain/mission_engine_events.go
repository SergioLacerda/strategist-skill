package domain

// MissionEngineEvent is the single event vocabulary accepted by the mission
// level transition boundary.
type MissionEngineEvent string

const (
	// MissionEventBootstrapDone signals bootstrap completion.
	MissionEventBootstrapDone MissionEngineEvent = "bootstrap_done"
	// MissionEventIntakeDone signals intake completion.
	MissionEventIntakeDone MissionEngineEvent = "intake_done"
	// MissionEventDiscoveryDone signals discovery completion.
	MissionEventDiscoveryDone MissionEngineEvent = "discovery_done"
	// MissionEventRefinementDone signals refinement with tasks.
	MissionEventRefinementDone MissionEngineEvent = "refinement_done"
	// MissionEventNoTasks signals refinement without delivery tasks.
	MissionEventNoTasks MissionEngineEvent = "refinement_done_no_tasks"
	// MissionEventGateApproved approves the current gate.
	MissionEventGateApproved MissionEngineEvent = "gate_approved"
	// MissionEventGateApprovedAnalysisOnly accepts a package that has no
	// documentation_target; the mission ends as analysis delivered.
	MissionEventGateApprovedAnalysisOnly MissionEngineEvent = "gate_approved_analysis_only"
	// MissionEventGateDenied denies the current gate.
	MissionEventGateDenied MissionEngineEvent = "gate_denied"
	// MissionEventGateTimeout closes an expired gate.
	MissionEventGateTimeout MissionEngineEvent = "gate_timeout"
	// MissionEventGateRevision requests refinement again.
	MissionEventGateRevision MissionEngineEvent = "gate_revision_requested"
	// MissionEventHandoffPassed is emitted only after the challenge result has
	// been persisted successfully at the live Archivist->Sniper boundary.
	MissionEventHandoffPassed MissionEngineEvent = "handoff_challenge_passed"
	// MissionEventHandoffFailed returns a repairable handoff to Archivist.
	MissionEventHandoffFailed MissionEngineEvent = "handoff_challenge_failed"
	// MissionEventHandoffNotApplicable ends a mission whose accepted package has
	// no documentation_target but that entered the handoff challenge anyway.
	MissionEventHandoffNotApplicable MissionEngineEvent = "handoff_challenge_not_applicable"
	// MissionEventHandoffExhausted is the fail-closed terminal outcome.
	MissionEventHandoffExhausted MissionEngineEvent = "handoff_challenge_exhausted"
	// MissionEventSniperDone signals execution completion.
	MissionEventSniperDone MissionEngineEvent = "sniper_done"
	// MissionEventRetryOK resumes after a transient failure.
	MissionEventRetryOK MissionEngineEvent = "retry_ok"
	// MissionEventSlotTransient records a retryable slot failure.
	MissionEventSlotTransient MissionEngineEvent = "slot_transient_failure"
	// MissionEventSlotPermanent records a terminal slot failure.
	MissionEventSlotPermanent MissionEngineEvent = "slot_permanent_failure"
	// MissionEventADRCriterion signals that the ADR criterion was met.
	MissionEventADRCriterion MissionEngineEvent = "adr_criterion_met"
	// MissionEventADRApproved approves the ADR gate.
	MissionEventADRApproved MissionEngineEvent = "adr_approved"
	// MissionEventADRDeclined declines the ADR gate.
	MissionEventADRDeclined MissionEngineEvent = "adr_declined"
)

func missionTransitionEvent(event MissionEngineEvent) (TransitionEvent, bool) {
	transitions := map[MissionEngineEvent]TransitionEvent{
		MissionEventRefinementDone: EventArchivistTasks, MissionEventNoTasks: EventArchivistNoTasks,
		MissionEventGateApproved: EventGateApproved, MissionEventGateDenied: EventGateDenied,
		MissionEventGateTimeout: EventGateTimeout, MissionEventGateRevision: EventGateRevision,
		MissionEventGateApprovedAnalysisOnly: EventGateApprovedAnalysisOnly,
		MissionEventHandoffPassed:            EventHandoffPassed, MissionEventHandoffFailed: EventHandoffFailed,
		MissionEventHandoffExhausted: EventHandoffExhausted, MissionEventHandoffNotApplicable: EventHandoffNotApplicable,
		MissionEventSniperDone: EventSniperDone, MissionEventRetryOK: EventRetryOK,
		MissionEventSlotTransient: EventSlotTransient, MissionEventSlotPermanent: EventSlotPermanent,
		MissionEventADRCriterion: EventADRCriterionMet, MissionEventADRApproved: EventADRApproved,
		MissionEventADRDeclined: EventADRDeclined,
	}
	value, ok := transitions[event]
	return value, ok
}

func phaseForState(state MissionState) PipelinePhase {
	switch state {
	case StateApprovalGate, StateHandoffChallenge, StateSideQuestGate, StateADRGate1, StateADRGate2, StateDirectGate:
		return PhaseApprovalGate
	case StateExecution, StateSideQuestExec, StateDirectExec, StateRetryingExecution, StateRetryingDirectExec:
		return PhaseExecution
	case StateDoneAnalysis, StateDoneDelivery, StateADRDone, StateDirectDone:
		return PhaseDone
	case StateBlocked:
		return PhaseBlocked
	case StateInit, StateSideQuestScan, StateRefinement, StateRetryingRefinement:
		return PhaseRefinement
	}
	return PhaseRefinement
}

// MissionEventRequiresNoDocumentationTargets reports the events that end a
// mission as analysis delivered after acceptance. They are valid only when the
// accepted package declares no documentation_target, which the submitting
// adapter checks against the refined tasks.md.
func MissionEventRequiresNoDocumentationTargets(event MissionEngineEvent) bool {
	return event == MissionEventGateApprovedAnalysisOnly || event == MissionEventHandoffNotApplicable
}

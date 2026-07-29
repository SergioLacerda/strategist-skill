package domain

// stateTransitions models gate/execution mechanics only: side-quest handling, the
// main Approval Gate, execution, retry-on-transient-failure, ADR, and
// Critical Hit. It intentionally does NOT model bootstrap, intake, discovery, or
// learning as states (S7) — those phases are enforced by contract + progress
// events, not by this transition table. Extending the FSM to the full pipeline is
// a separate design decision (interacts with W7's single-source compilation) and
// is out of scope here; see the "full-pipeline FSM" follow-up in
// .analysis/todo/analise-tecnica.md.
var stateTransitions = map[MissionState]map[TransitionEvent]MissionState{
	StateInit: {
		EventDirectHitIntent:  StateDirectGate,
		EventManifestEmpty:    StateSideQuestScan,
		EventManifestNonEmpty: StateSideQuestScan,
	},
	StateSideQuestScan: {
		EventManifestEmpty:    StateRefinement,
		EventManifestNonEmpty: StateSideQuestGate,
	},
	StateSideQuestGate: {
		EventGateDenied:   StateRefinement,
		EventGateApproved: StateSideQuestExec,
	},
	StateSideQuestExec: {
		EventSniperDone: StateRefinement,
	},
	StateRefinement: {
		EventArchivistNoTasks: StateDoneAnalysis,
		EventArchivistTasks:   StateApprovalGate,
		EventSlotTransient:    StateRetryingRefinement,
		EventSlotPermanent:    StateBlocked,
	},
	StateApprovalGate: {
		EventGateDenied:   StateDoneAnalysis,
		EventGateApproved: StateExecution,
		EventGateTimeout:  StateDoneAnalysis,
		EventGateRevision: StateRefinement, // D2: documented revision loop, now representable
	},
	StateExecution: {
		EventSniperDone:      StateDoneDelivery,
		EventSniperSideQuest: StateSideQuestGate,
		EventSlotTransient:   StateRetryingExecution,
		EventSlotPermanent:   StateBlocked,
	},
	StateDoneAnalysis: {
		EventADRCriterionMet: StateADRGate1,
	},
	StateDoneDelivery: {
		EventADRCriterionMet: StateADRGate1,
	},
	StateRetryingRefinement: {
		EventRetryOK:       StateRefinement,
		EventSlotPermanent: StateBlocked,
		EventSlotTransient: StateBlocked,
	},
	StateRetryingExecution: {
		EventRetryOK:       StateExecution,
		EventSlotPermanent: StateBlocked,
		EventSlotTransient: StateBlocked,
	},
	StateADRGate1: {
		EventADRApproved: StateADRGate2,
		EventADRDeclined: StateADRDone,
	},
	StateADRGate2: {
		EventADRApproved: StateADRDone,
		EventADRDeclined: StateADRDone,
	},
	StateDirectGate: {
		EventDirectGateApproved: StateDirectExec,
		EventDirectGateDeclined: StateDoneAnalysis,
	},
	StateDirectExec: {
		EventSniperDone:    StateDirectDone,
		EventSlotTransient: StateRetryingDirectExec,
		EventSlotPermanent: StateBlocked,
	},
	StateRetryingDirectExec: {
		EventRetryOK:       StateDirectExec,
		EventSlotPermanent: StateBlocked,
		EventSlotTransient: StateBlocked,
	},
}

// NextState applies a single transition event to the current state using a predefined transition table.
// Unhandled events for a given state return the current state (a self-loop).
func NextState(current MissionState, event TransitionEvent) MissionState {
	if transitions, ok := stateTransitions[current]; ok {
		if next, ok := transitions[event]; ok {
			return next
		}
	}
	return current
}

// RunStateMachine folds a sequence of events from a starting state.
func RunStateMachine(start MissionState, events []TransitionEvent) MissionState {
	state := start
	for _, ev := range events {
		state = NextState(state, ev)
	}
	return state
}

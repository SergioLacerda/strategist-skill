package domain

var stateTransitions = map[MissionState]map[TransitionEvent]MissionState{
	StateInit: {
		EventQuickDrawIntent:  StateQuickDraw,
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
	StateRetrying: {
		EventManifestNonEmpty: StateRefinement,
		EventSlotPermanent:    StateBlocked,
		EventSlotTransient:    StateBlocked,
	},
	StateRetryingRefinement: {
		EventManifestNonEmpty: StateRefinement,
		EventSlotPermanent:    StateBlocked,
		EventSlotTransient:    StateBlocked,
	},
	StateRetryingExecution: {
		EventManifestNonEmpty: StateExecution,
		EventSlotPermanent:    StateBlocked,
		EventSlotTransient:    StateBlocked,
	},
	StateQuickDraw: {
		EventManifestNonEmpty: StateQuickDrawGate,
		EventManifestEmpty:    StateQuickDrawDone,
	},
	StateQuickDrawGate: {
		EventQuickDrawApprove: StateQuickDrawDone,
		EventQuickDrawDecline: StateQuickDrawDone,
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
		EventManifestNonEmpty: StateDirectExec,
		EventSlotPermanent:    StateBlocked,
		EventSlotTransient:    StateBlocked,
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

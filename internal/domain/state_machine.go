package domain

// NextState applies a single transition event to the current state using policy guards.
func NextState(current MissionState, event TransitionEvent, policy MissionPolicy) MissionState {
	p := NormalizePolicy(policy)

	switch current {
	case StateInit:
		return nextFromInit(event)
	case StateOpportunityAttack:
		return nextFromOpportunityAttack(event)
	case StateOpportunityGate:
		return nextFromOpportunityGate(event, p)
	case StateOpportunityExec:
		return nextFromOpportunityExec(event)
	case StateRefinement:
		return nextFromRefinement(event, p)
	case StateApprovalGate:
		return nextFromApprovalGate(event, p)
	case StateExecution:
		return nextFromExecution(event)
	case StateRetrying:
		return nextFromRetrying(event, p)
	case StateQuickDraw:
		return nextFromQuickDraw(event)
	case StateQuickDrawGate:
		return nextFromQuickDrawGate(event)
	case StateADRGate1:
		return nextFromADRGate1(event)
	case StateADRGate2:
		return nextFromADRGate2(event)
	case StateDoneAnalysis:
		return nextFromDoneAnalysis(event)
	case StateDoneDelivery:
		return nextFromDoneDelivery(event)
	case StateQuickDrawDone, StateADRDone, StateBlocked:
		return current
	}

	return current
}

func nextFromInit(event TransitionEvent) MissionState {
	switch event {
	case EventQuickDrawIntent:
		return StateQuickDraw
	case EventManifestEmpty, EventManifestNonEmpty:
		return StateOpportunityAttack
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateInit
	}
	return StateInit
}

func nextFromOpportunityAttack(event TransitionEvent) MissionState {
	switch event {
	case EventManifestEmpty:
		return StateRefinement
	case EventManifestNonEmpty:
		return StateOpportunityGate
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateOpportunityAttack
	}
	return StateOpportunityAttack
}

func nextFromOpportunityGate(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventGateDenied:
		return StateRefinement
	case EventGateApproved:
		if p.CanExecute {
			return StateOpportunityExec
		}
		return StateRefinement
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateOpportunityGate
	}
	return StateOpportunityGate
}

func nextFromOpportunityExec(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateRefinement
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateOpportunityExec
	}
	return StateOpportunityExec
}

func nextFromRefinement(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventArchivistNoTasks:
		return StateDoneAnalysis
	case EventArchivistTasks:
		if p.Mode == MissionModeAnalysis {
			return StateDoneAnalysis
		}
		return StateApprovalGate
	case EventSlotTransient:
		return StateRetrying
	case EventSlotPermanent:
		return StateBlocked
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined, EventSniperOA:
		return StateRefinement
	}
	return StateRefinement
}

func nextFromApprovalGate(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventGateDenied:
		return StateDoneAnalysis
	case EventGateApproved:
		if p.CanExecute {
			return StateExecution
		}
		return StateDoneDelivery
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateApprovalGate
	}
	return StateApprovalGate
}

func nextFromExecution(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateDoneDelivery
	case EventSniperOA:
		// Mid-execution opportunity attack surfaced — pause for user review.
		return StateOpportunityGate
	case EventSlotTransient:
		return StateRetrying
	case EventSlotPermanent:
		return StateBlocked
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined:
		return StateExecution
	}
	return StateExecution
}

func nextFromDoneAnalysis(event TransitionEvent) MissionState {
	if event == EventADRCriterionMet {
		return StateADRGate1
	}
	return StateDoneAnalysis
}

func nextFromDoneDelivery(event TransitionEvent) MissionState {
	if event == EventADRCriterionMet {
		return StateADRGate1
	}
	return StateDoneDelivery
}

// nextFromRetrying handles a single transient retry attempt.
// A second transient or any permanent failure always goes to BLOCKED.
func nextFromRetrying(event TransitionEvent, _ MissionPolicy) MissionState {
	switch event {
	case EventManifestNonEmpty:
		return StateRefinement
	case EventSlotPermanent, EventSlotTransient:
		return StateBlocked
	case EventManifestEmpty, EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined, EventSniperOA:
		return StateRetrying
	}
	return StateRetrying
}

func nextFromQuickDraw(event TransitionEvent) MissionState {
	switch event {
	case EventManifestNonEmpty:
		return StateQuickDrawGate
	case EventManifestEmpty:
		return StateQuickDrawDone
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateQuickDraw
	}
	return StateQuickDraw
}

func nextFromQuickDrawGate(event TransitionEvent) MissionState {
	switch event {
	case EventQuickDrawApprove, EventQuickDrawDecline:
		return StateQuickDrawDone
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone,
		EventArchivistNoTasks, EventArchivistTasks, EventQuickDrawIntent,
		EventADRCriterionMet, EventADRApproved, EventADRDeclined,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateQuickDrawGate
	}
	return StateQuickDrawGate
}

func nextFromADRGate1(event TransitionEvent) MissionState {
	switch event {
	case EventADRApproved:
		return StateADRGate2
	case EventADRDeclined:
		return StateADRDone
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone,
		EventArchivistNoTasks, EventArchivistTasks, EventADRCriterionMet,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateADRGate1
	}
	return StateADRGate1
}

func nextFromADRGate2(event TransitionEvent) MissionState {
	switch event {
	case EventADRApproved, EventADRDeclined:
		return StateADRDone
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone,
		EventArchivistNoTasks, EventArchivistTasks, EventADRCriterionMet,
		EventQuickDrawIntent, EventQuickDrawApprove, EventQuickDrawDecline,
		EventSlotTransient, EventSlotPermanent, EventSniperOA:
		return StateADRGate2
	}
	return StateADRGate2
}

// RunStateMachine folds a sequence of events from a starting state.
func RunStateMachine(start MissionState, events []TransitionEvent, policy MissionPolicy) MissionState {
	state := start
	for _, ev := range events {
		state = NextState(state, ev, policy)
	}
	return state
}

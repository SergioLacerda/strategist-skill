package domain

// NextState applies a single transition event to the current state using policy guards.
func NextState(current MissionState, event TransitionEvent, policy MissionPolicy) MissionState {
	p := NormalizePolicy(policy)

	switch current {
	case StateInit:
		return nextFromInit(event)
	case StateHousekeeping:
		return nextFromHousekeeping(event)
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
	case StateDoneAnalise:
		return StateDoneAnalise
	case StateDoneEntrega:
		return StateDoneEntrega
	case StateBlocked:
		return StateBlocked
	}

	return current
}

func nextFromInit(event TransitionEvent) MissionState {
	switch event {
	case EventManifestEmpty, EventManifestNonEmpty:
		return StateHousekeeping
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateInit
	}
	return StateInit
}

func nextFromHousekeeping(event TransitionEvent) MissionState {
	switch event {
	case EventManifestEmpty:
		return StateRefinement
	case EventManifestNonEmpty:
		return StateOpportunityGate
	case EventGateApproved, EventGateDenied, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateHousekeeping
	}
	return StateHousekeeping
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
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateOpportunityGate
	}
	return StateOpportunityGate
}

func nextFromOpportunityExec(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateRefinement
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks:
		return StateOpportunityExec
	}
	return StateOpportunityExec
}

func nextFromRefinement(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventArchivistNoTasks:
		return StateDoneAnalise
	case EventArchivistTasks:
		if p.Mode == MissionModeAnalise {
			return StateDoneAnalise
		}
		return StateApprovalGate
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventSniperDone:
		return StateRefinement
	}
	return StateRefinement
}

func nextFromApprovalGate(event TransitionEvent, p MissionPolicy) MissionState {
	switch event {
	case EventGateDenied:
		return StateDoneAnalise
	case EventGateApproved:
		if p.CanExecute {
			return StateExecution
		}
		return StateDoneEntrega
	case EventManifestEmpty, EventManifestNonEmpty, EventSniperDone, EventArchivistNoTasks, EventArchivistTasks:
		return StateApprovalGate
	}
	return StateApprovalGate
}

func nextFromExecution(event TransitionEvent) MissionState {
	switch event {
	case EventSniperDone:
		return StateDoneEntrega
	case EventManifestEmpty, EventManifestNonEmpty, EventGateApproved, EventGateDenied, EventArchivistNoTasks, EventArchivistTasks:
		return StateExecution
	}
	return StateExecution
}

// RunStateMachine folds a sequence of events from a starting state.
func RunStateMachine(start MissionState, events []TransitionEvent, policy MissionPolicy) MissionState {
	state := start
	for _, ev := range events {
		state = NextState(state, ev, policy)
	}
	return state
}

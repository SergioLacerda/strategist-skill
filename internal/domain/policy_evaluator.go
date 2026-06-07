package domain

// EvaluateGuardedTransition applies mission policy + approval gate status.
// Invariants:
// - Any guarded transition requires explicit approval.
// - Execution transitions require can_execute=true.
func EvaluateGuardedTransition(policy MissionPolicy, transitionGroup string, gateApproved bool) TransitionDecision {
	p := NormalizePolicy(policy)
	if err := p.Validate(); err != nil {
		return TransitionDecision{Allowed: false, Reason: err.Error(), Status: "policy_blocked", Policy: p}
	}

	if !gateApproved {
		return TransitionDecision{Allowed: false, Reason: "approval_required", Status: "approval_required", Policy: p}
	}

	switch transitionGroup {
	case TransitionGroupExecution:
		if !p.CanExecute {
			return TransitionDecision{Allowed: false, Reason: "policy_blocked", Status: "policy_blocked", Policy: p}
		}
		return TransitionDecision{Allowed: true, Reason: "allowed", Status: "allowed", Policy: p}
	case TransitionGroupFinalizeAnalysis:
		return TransitionDecision{Allowed: true, Reason: "allowed", Status: "allowed", Policy: p}
	default:
		return TransitionDecision{Allowed: false, Reason: "unknown_transition_group", Status: "policy_blocked", Policy: p}
	}
}

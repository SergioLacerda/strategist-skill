package domain

// EvaluateGuardedTransition applies review gate status to guarded transitions.
// Invariants:
// - Any guarded transition requires explicit review gate acceptance.
// - Documentation materialization is always permitted after review gate acceptance.
func EvaluateGuardedTransition(policy MissionPolicy, transitionGroup string, gateApproved bool) TransitionDecision {
	p := NormalizePolicy(policy)

	if !gateApproved {
		return TransitionDecision{Allowed: false, Reason: "approval_required", Status: "approval_required", Policy: p}
	}

	switch transitionGroup {
	case TransitionGroupDocumentation, TransitionGroupReviewGate, TransitionGroupFinalizeAnalysis:
		return TransitionDecision{Allowed: true, Reason: "allowed", Status: "allowed", Policy: p}
	default:
		return TransitionDecision{Allowed: false, Reason: "unknown_transition_group", Status: "policy_blocked", Policy: p}
	}
}

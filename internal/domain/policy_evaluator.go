package domain

// EvaluateGuardedTransition applies review gate status to guarded transitions.
// Invariants:
// - Any guarded transition requires explicit review gate acceptance.
// - Documentation materialization is always permitted after review gate acceptance.
func EvaluateGuardedTransition(transitionGroup string, gateApproved bool) TransitionDecision {
	if !gateApproved {
		return TransitionDecision{Allowed: false, Reason: PolicyReasonApprovalRequired, Status: PolicyStatusApprovalRequired}
	}

	switch transitionGroup {
	case TransitionGroupDocumentation, TransitionGroupReviewGate, TransitionGroupFinalizeAnalysis:
		return TransitionDecision{Allowed: true, Reason: PolicyReasonAllowed, Status: PolicyStatusAllowed}
	default:
		return TransitionDecision{Allowed: false, Reason: PolicyReasonUnknownTransitionGroup, Status: PolicyStatusBlocked}
	}
}

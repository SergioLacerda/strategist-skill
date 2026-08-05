package handoff

// Policy decides whether a semantic handoff challenge is required for a transition.
// It is intentionally data-only so contracts, fixtures, and runtime code can share
// the same vocabulary without depending on provider invocation.
type Policy struct {
	Enabled            bool
	Transition         string
	RequiredTypes      []string
	RequireAllCritical bool
	MaxAttempts        int
	OnFailure          string
	// ForbiddenClaims lists claims the acknowledgment must never assert,
	// independent of which challenges were generated — a policy-level
	// safety net, not tied to a specific Challenge. Each entry is either
	// ForbiddenClaimExecutionAuthorized, or "<ref>_as_approved" (checked
	// against Acknowledgment.Classifications). Optional; nil means no
	// forbidden-claim checking beyond what individual challenges already
	// assert via ExpectedClassification/ExpectedGateAllowed.
	ForbiddenClaims []string
}

// RiskSignals describe handoff traits that make a challenge mandatory.
type RiskSignals struct {
	ApprovalGatePresent          bool
	MandatoryConstraintsPresent  bool
	UnresolvedQuestionsPresent   bool
	ForbiddenScopePresent        bool
	ImplementationHandoffPresent bool
	DestructiveOperationPossible bool
	SecuritySensitiveTask        bool
	InformationalOnly            bool
}

// DefaultPolicy returns the Handoff Challenge MVP policy
// (TransitionArchivistToSniper).
func DefaultPolicy() Policy {
	return Policy{
		Enabled:            true,
		Transition:         TransitionArchivistToSniper,
		RequiredTypes:      []string{ChallengeObjective, ChallengeBoundary, ChallengeClassification, ChallengeGate},
		RequireAllCritical: true,
		MaxAttempts:        2,
		OnFailure:          FailureActionReturnToArchivist,
	}
}

// RangerToArchivistPolicy returns the built-in policy for the
// Ranger->Archivist transition, per
// .analysis/refined/20260803-handoff-challenge-extensions/design.md § Item
// 1 — advisory-first (Enabled: false by default; RequiredByRisk still
// applies for callers that want risk-based activation instead of a fixed
// Enabled value).
func RangerToArchivistPolicy() Policy {
	return Policy{
		Enabled:            false,
		Transition:         TransitionRangerToArchivist,
		RequiredTypes:      []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict},
		RequireAllCritical: true,
		MaxAttempts:        2,
		OnFailure:          FailureActionReturnToArchivist,
	}
}

// SniperToValidationPolicy returns the built-in policy for the
// Sniper->validation transition (quiz.txt's third proposed integration
// point). Advisory-first, same posture as RangerToArchivistPolicy — no
// consuming role currently sets this required by default.
func SniperToValidationPolicy() Policy {
	return Policy{
		Enabled:            false,
		Transition:         TransitionSniperToValidation,
		RequiredTypes:      []string{ChallengeBoundary, ChallengeClassification},
		RequireAllCritical: true,
		MaxAttempts:        2,
		OnFailure:          FailureActionReturnToArchivist,
	}
}

// RequiredByRisk reports whether risk signals require a challenge.
func RequiredByRisk(s RiskSignals) bool {
	if s.InformationalOnly && !s.ApprovalGatePresent && !s.MandatoryConstraintsPresent &&
		!s.UnresolvedQuestionsPresent && !s.ForbiddenScopePresent &&
		!s.ImplementationHandoffPresent && !s.DestructiveOperationPossible &&
		!s.SecuritySensitiveTask {
		return false
	}
	return s.ApprovalGatePresent ||
		s.MandatoryConstraintsPresent ||
		s.UnresolvedQuestionsPresent ||
		s.ForbiddenScopePresent ||
		s.ImplementationHandoffPresent ||
		s.DestructiveOperationPossible ||
		s.SecuritySensitiveTask
}

// StatusForRisk returns the policy status implied by risk signals.
func StatusForRisk(s RiskSignals) string {
	if RequiredByRisk(s) {
		return StatusRequired
	}
	return StatusSkipped
}

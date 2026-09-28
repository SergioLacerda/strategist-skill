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
	RequireWhen        []PolicyPredicate
	SkipWhen           []PolicyPredicate
	RequirePrecedence  bool
	// ForbiddenClaims lists claims the acknowledgment must never assert,
	// independent of which challenges were generated — a policy-level
	// safety net, not tied to a specific Challenge. Each entry is either
	// ForbiddenClaimExecutionAuthorized, or "<ref>_as_approved" (checked
	// against Acknowledgment.Classifications). Optional; nil means no
	// forbidden-claim checking beyond what individual challenges already
	// assert via ExpectedClassification/ExpectedGateAllowed.
	ForbiddenClaims []string
}

// PolicyPredicate is a machine-evaluated condition from a handoff policy
// contract. RequireWhen is an OR list; SkipWhen is an AND list. If both match,
// require precedence keeps a risk-bearing handoff fail-closed.
type PolicyPredicate string

// Predicate values a handoff policy contract's RequireWhen/SkipWhen lists reference.
const (
	PredicateMandatoryConstraintsPresent          PolicyPredicate = "mandatory_constraints_present"
	PredicateUnresolvedQuestionsPresent           PolicyPredicate = "unresolved_questions_present"
	PredicateForbiddenScopePresent                PolicyPredicate = "forbidden_scope_present"
	PredicateImplementationHandoffPresent         PolicyPredicate = "implementation_handoff_present"
	PredicateDestructiveOperationPossible         PolicyPredicate = "destructive_operation_possible"
	PredicateSecuritySensitiveTask                PolicyPredicate = "security_sensitive_task"
	PredicateInformationalOnly                    PolicyPredicate = "informational_only"
	PredicateNoCriticalConstraints                PolicyPredicate = "no_critical_constraints"
	PredicateNoUnresolvedQuestions                PolicyPredicate = "no_unresolved_questions"
	PredicateNoForbiddenScopeBeyondSniperDefaults PolicyPredicate = "no_forbidden_scope_beyond_sniper_defaults"
)

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
		RequireWhen: []PolicyPredicate{
			PredicateMandatoryConstraintsPresent,
			PredicateUnresolvedQuestionsPresent,
			PredicateForbiddenScopePresent,
			PredicateImplementationHandoffPresent,
			PredicateDestructiveOperationPossible,
			PredicateSecuritySensitiveTask,
		},
		SkipWhen: []PolicyPredicate{
			PredicateInformationalOnly,
			PredicateNoCriticalConstraints,
			PredicateNoUnresolvedQuestions,
			PredicateNoForbiddenScopeBeyondSniperDefaults,
		},
		RequirePrecedence: true,
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

// RiskSignalsForLevel maps a mission's coarse risk_level classification —
// "low", "medium", or "high", the shape produced by the prompt-intake skill
// (.strategist/internal_skills/prompt-intake/skill.yaml: "risk_level:
// string # low, medium, high") and carried on mission state from intake —
// onto the RiskSignals vocabulary RequiredByRisk/StatusForRisk already
// understand.
//
// Mission state at the call site only carries the coarse label, not
// individual signal flags, so this is a deliberately conservative
// approximation rather than a precise re-derivation of the underlying
// signals: "low" maps to InformationalOnly (skipped, matching
// handoff-contract.yaml's skip_when); "medium" (MandatoryConstraintsPresent)
// and "high" (ApprovalGatePresent/DestructiveOperationPossible/
// SecuritySensitiveTask) each set at least one of RequiredByRisk's signal
// fields, and RequiredByRisk is a plain OR across all of them — so today
// "medium" and "high" both resolve to required, with the same RequiredTypes
// (riskGatedPolicy does not vary RequiredTypes by level). There is
// currently no behavioral distinction between the two tiers; a future
// refinement could give "medium" a narrower RequiredTypes subset or
// different MaxAttempts if that distinction turns out to matter in
// practice (see .analysis/refined/20260830-skill-gaps-followup/design.md
// § F2's Option 2). An unrecognized or empty level maps to the zero value,
// which RequiredByRisk treats as not required — advisory-first still holds
// when risk is unknown rather than failing closed into a mandatory
// challenge.
func RiskSignalsForLevel(riskLevel string) RiskSignals {
	switch riskLevel {
	case "high":
		return RiskSignals{
			ApprovalGatePresent:          true,
			DestructiveOperationPossible: true,
			SecuritySensitiveTask:        true,
		}
	case "medium":
		return RiskSignals{MandatoryConstraintsPresent: true}
	case "low":
		return RiskSignals{InformationalOnly: true}
	default:
		return RiskSignals{}
	}
}

// ResolvePolicyForMission, riskGatedPolicy, and the predicate-matching
// helpers live in policy_risk_gate.go, split out to keep this file under the
// repo's file-size budget.

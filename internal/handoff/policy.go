package handoff

import "fmt"

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

// ResolvePolicyForMission builds the handoff policy for transition with
// Enabled/RequiredTypes driven by the mission's actual risk_level, instead
// of the fixed advisory-first Enabled: false baked into
// RangerToArchivistPolicy and SniperToValidationPolicy. This is the
// missing caller identified by
// .analysis/refined/20260830-skill-gaps-triage/analysis.md Cluster 11
// (K22): RequiredByRisk/StatusForRisk already existed but nothing invoked
// them against real mission state.
//
// TransitionArchivistToSniper is unaffected by riskLevel: per
// handoff-contract.yaml's handoff_verification_policy.default_policy, it
// is Strategist's MVP challenge and stays required by default regardless
// of risk classification; RangerToArchivistPolicy and
// SniperToValidationPolicy are advisory-first extensions (see their doc
// comments) that this function activates only when the mission's risk
// signals warrant it.
//
// Returns an error for a transition string that isn't one of the three
// known constants, so callers can distinguish "risk resolution ran and
// found nothing required" from "transition not recognized" — a zero-value
// Policy would silently look like the former.
func ResolvePolicyForMission(riskLevel, transition string) (Policy, error) {
	switch transition {
	case TransitionArchivistToSniper:
		return DefaultPolicy(), nil
	case TransitionRangerToArchivist:
		return riskGatedPolicy(RangerToArchivistPolicy(), riskLevel), nil
	case TransitionSniperToValidation:
		return riskGatedPolicy(SniperToValidationPolicy(), riskLevel), nil
	default:
		return Policy{}, fmt.Errorf("handoff: unknown transition %q (want %s, %s, or %s)",
			transition, TransitionArchivistToSniper, TransitionRangerToArchivist, TransitionSniperToValidation)
	}
}

// riskGatedPolicy flips base.Enabled on/off per riskLevel's derived risk
// signals, clearing RequiredTypes when the challenge isn't required so a
// skipped policy doesn't advertise required types it will never enforce
// (Verify already short-circuits on !Enabled, but a cleared list keeps the
// returned Policy value self-consistent for callers that inspect it
// directly, e.g. to log or serialize).
func riskGatedPolicy(base Policy, riskLevel string) Policy {
	base.Enabled = StatusForRisk(RiskSignalsForLevel(riskLevel)) == StatusRequired
	if !base.Enabled {
		base.RequiredTypes = nil
	}
	return base
}

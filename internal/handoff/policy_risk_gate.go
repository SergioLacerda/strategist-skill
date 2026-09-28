package handoff

import "fmt"

// ResolvePolicyForMission builds the handoff policy for transition with
// Enabled/RequiredTypes driven by the mission's actual risk_level, instead
// of the fixed advisory-first Enabled: false baked into
// RangerToArchivistPolicy and SniperToValidationPolicy. This is the
// missing caller identified by
// .analysis/refined/20260830-skill-gaps-triage/analysis.md Cluster 11
// (K22): RequiredByRisk/StatusForRisk already existed but nothing invoked
// them against real mission state.
//
// TransitionArchivistToSniper uses the contract's low-risk skip predicates:
// informational missions may skip only the semantic challenge, while any
// matching risk predicate requires it. The Approval Gate remains mandatory
// independently of this policy. RangerToArchivistPolicy and
// SniperToValidationPolicy remain advisory-first extensions (see their doc
// comments) that this function activates when the mission's risk signals
// warrant it.
//
// Returns an error for a transition string that isn't one of the three
// known constants, so callers can distinguish "risk resolution ran and
// found nothing required" from "transition not recognized" — a zero-value
// Policy would silently look like the former.
func ResolvePolicyForMission(riskLevel, transition string) (Policy, error) {
	switch transition {
	case TransitionArchivistToSniper:
		return riskGatedPolicy(DefaultPolicy(), riskLevel), nil
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
	if len(base.RequireWhen) == 0 && len(base.SkipWhen) == 0 {
		base.Enabled = StatusForRisk(RiskSignalsForLevel(riskLevel)) == StatusRequired
	} else {
		base.Enabled = statusForPolicy(base, RiskSignalsForLevel(riskLevel)) == StatusRequired
	}
	if !base.Enabled {
		base.RequiredTypes = nil
	}
	return base
}

func statusForPolicy(policy Policy, signals RiskSignals) string {
	if anyPredicateMatches(policy.RequireWhen, signals) {
		return StatusRequired
	}
	if len(policy.SkipWhen) > 0 && allPredicatesMatch(policy.SkipWhen, signals) {
		return StatusSkipped
	}
	return StatusRequired
}

func anyPredicateMatches(predicates []PolicyPredicate, signals RiskSignals) bool {
	for _, predicate := range predicates {
		if predicateMatches(predicate, signals) {
			return true
		}
	}
	return false
}

func allPredicatesMatch(predicates []PolicyPredicate, signals RiskSignals) bool {
	for _, predicate := range predicates {
		if !predicateMatches(predicate, signals) {
			return false
		}
	}
	return true
}

func predicateMatches(predicate PolicyPredicate, signals RiskSignals) bool {
	switch predicate {
	case PredicateMandatoryConstraintsPresent:
		return signals.MandatoryConstraintsPresent
	case PredicateUnresolvedQuestionsPresent:
		return signals.UnresolvedQuestionsPresent
	case PredicateForbiddenScopePresent:
		return signals.ForbiddenScopePresent
	case PredicateImplementationHandoffPresent:
		return signals.ImplementationHandoffPresent
	case PredicateDestructiveOperationPossible:
		return signals.DestructiveOperationPossible
	case PredicateSecuritySensitiveTask:
		return signals.SecuritySensitiveTask
	case PredicateInformationalOnly:
		return signals.InformationalOnly
	case PredicateNoCriticalConstraints:
		return !signals.MandatoryConstraintsPresent
	case PredicateNoUnresolvedQuestions:
		return !signals.UnresolvedQuestionsPresent
	case PredicateNoForbiddenScopeBeyondSniperDefaults:
		return !signals.ForbiddenScopePresent
	default:
		return false
	}
}

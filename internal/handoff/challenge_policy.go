package handoff

import (
	"errors"
	"fmt"
	"strings"
)

// allowedChallengeTypesByTransition scopes valid challenge types per
// transition — the MVP's four types stay valid only for
// TransitionArchivistToSniper, and TransitionRangerToArchivist has its own
// four, so a challenge type with no referent on a given transition (e.g.
// "gate" on a Ranger handoff, which never has an approval gate) is rejected
// by ValidatePolicy instead of silently accepted.
var allowedChallengeTypesByTransition = map[string]map[string]bool{
	TransitionArchivistToSniper: {
		ChallengeObjective:      true,
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeGate:           true,
		ChallengeCounterfactual: true,
	},
	TransitionRangerToArchivist: {
		ChallengeRecall:         true,
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeVerdict:        true,
	},
	TransitionSniperToValidation: {
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeCounterfactual: true,
	},
}

// ValidatePolicy verifies that a policy is usable by the MVP verifier.
func ValidatePolicy(p Policy) error {
	var errs []error
	errs = append(errs, validateTransition(p.Transition)...)
	errs = append(errs, validateRequiredTypesPresence(p)...)
	errs = append(errs, validateChallengeTypes(p.Transition, p.RequiredTypes)...)
	errs = append(errs, validateMaxAttempts(p.MaxAttempts)...)
	errs = append(errs, validateOnFailure(p.OnFailure)...)
	errs = append(errs, validatePredicates(p.RequireWhen, p.SkipWhen, p.RequirePrecedence)...)
	errs = append(errs, validateForbiddenClaims(p.ForbiddenClaims)...)
	return errors.Join(errs...)
}

var supportedPolicyPredicates = map[PolicyPredicate]struct{}{
	PredicateMandatoryConstraintsPresent:          {},
	PredicateUnresolvedQuestionsPresent:           {},
	PredicateForbiddenScopePresent:                {},
	PredicateImplementationHandoffPresent:         {},
	PredicateDestructiveOperationPossible:         {},
	PredicateSecuritySensitiveTask:                {},
	PredicateInformationalOnly:                    {},
	PredicateNoCriticalConstraints:                {},
	PredicateNoUnresolvedQuestions:                {},
	PredicateNoForbiddenScopeBeyondSniperDefaults: {},
}

func validatePredicates(requireWhen, skipWhen []PolicyPredicate, requirePrecedence bool) []error {
	requireSet, errs := validateRequireWhenPredicates(requireWhen)
	errs = append(errs, validateSkipWhenPredicates(skipWhen, requireSet)...)
	if len(requireWhen) > 0 && len(skipWhen) > 0 && !requirePrecedence {
		errs = append(errs, errors.New("handoff_policy_invalid: require precedence is required when require_when and skip_when are both configured"))
	}
	return errs
}

// validateRequireWhenPredicates rejects unknown predicates and returns the
// set of declared ones, so validateSkipWhenPredicates can detect overlap.
func validateRequireWhenPredicates(requireWhen []PolicyPredicate) (map[PolicyPredicate]struct{}, []error) {
	var errs []error
	requireSet := make(map[PolicyPredicate]struct{}, len(requireWhen))
	for _, predicate := range requireWhen {
		if _, ok := supportedPolicyPredicates[predicate]; !ok {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: unknown require_when predicate %q", predicate))
		}
		requireSet[predicate] = struct{}{}
	}
	return requireSet, errs
}

// validateSkipWhenPredicates rejects unknown predicates and any predicate
// that also appears in require_when (an OR/AND list overlap is ambiguous).
func validateSkipWhenPredicates(skipWhen []PolicyPredicate, requireSet map[PolicyPredicate]struct{}) []error {
	var errs []error
	for _, predicate := range skipWhen {
		if _, ok := supportedPolicyPredicates[predicate]; !ok {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: unknown skip_when predicate %q", predicate))
		}
		if _, ok := requireSet[predicate]; ok {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: predicate %q cannot appear in both require_when and skip_when", predicate))
		}
	}
	return errs
}

// validateForbiddenClaims rejects a ForbiddenClaims entry that matches
// neither recognized form, so a typo (e.g. "Q-01_is_approved") fails fast
// at policy-validation time instead of silently never matching in Verify.
func validateForbiddenClaims(claims []string) []error {
	var errs []error
	for _, claim := range claims {
		if claim == ForbiddenClaimExecutionAuthorized {
			continue
		}
		if strings.HasSuffix(claim, forbiddenClaimAsApprovedSuffix) && len(claim) > len(forbiddenClaimAsApprovedSuffix) {
			continue
		}
		errs = append(errs, fmt.Errorf("handoff_policy_invalid: forbidden_claims entry %q is not recognized (want %q or \"<ref>%s\")",
			claim, ForbiddenClaimExecutionAuthorized, forbiddenClaimAsApprovedSuffix))
	}
	return errs
}

func validateTransition(transition string) []error {
	if transition == "" {
		return []error{errors.New("handoff_policy_invalid: transition is required")}
	}
	if _, ok := allowedChallengeTypesByTransition[transition]; !ok {
		return []error{fmt.Errorf("handoff_policy_invalid: transition %q is not supported", transition)}
	}
	return nil
}

func validateRequiredTypesPresence(p Policy) []error {
	if p.Enabled && len(p.RequiredTypes) == 0 {
		return []error{errors.New("handoff_policy_invalid: required_types is required when enabled")}
	}
	return nil
}

func validateChallengeTypes(transition string, types []string) []error {
	allowed := allowedChallengeTypesByTransition[transition]
	var errs []error
	for _, typ := range types {
		if !allowed[typ] {
			errs = append(errs, fmt.Errorf("handoff_policy_invalid: challenge type %q is not allowed for transition %q", typ, transition))
		}
	}
	return errs
}

func validateMaxAttempts(maxAttempts int) []error {
	if maxAttempts < 0 {
		return []error{errors.New("handoff_policy_invalid: max_attempts must be >= 0")}
	}
	return nil
}

func validateOnFailure(onFailure string) []error {
	if onFailure != "" && onFailure != FailureActionReturnToArchivist {
		return []error{fmt.Errorf("handoff_policy_invalid: on_failure %q is not supported by the MVP", onFailure)}
	}
	return nil
}

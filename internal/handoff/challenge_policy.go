package handoff

import (
	"errors"
	"fmt"
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
	},
	TransitionRangerToArchivist: {
		ChallengeRecall:         true,
		ChallengeBoundary:       true,
		ChallengeClassification: true,
		ChallengeVerdict:        true,
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
	return errors.Join(errs...)
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

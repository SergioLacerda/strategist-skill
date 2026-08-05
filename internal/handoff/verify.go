package handoff

import "strings"

// Verify evaluates Sniper acknowledgment against Archivist challenges.
func Verify(policy Policy, challenges []Challenge, ack Acknowledgment) Result {
	if err := ValidatePolicy(policy); err != nil {
		return failedResult(policy, []string{err.Error()}, nil, nil, false, nil, nil)
	}
	if !policy.Enabled {
		return Result{Status: StatusSkipped, Passed: true}
	}

	missingChallenges := missingRequiredTypes(policy.RequiredTypes, challenges)
	missingRefs := missingSourceRefs(challenges, ack.UnderstoodRefs)
	misclassified := misclassifiedRefs(challenges, ack.Classifications)
	gateMismatch := gateMismatch(challenges, ack.GateAllowed)
	counterfactualMismatches := counterfactualMismatches(challenges, ack.CounterfactualAnswers)
	forbiddenClaimViolations := forbiddenClaimViolations(policy.ForbiddenClaims, ack)

	result := failedResult(policy, missingRefs, missingChallenges, misclassified, gateMismatch, counterfactualMismatches, forbiddenClaimViolations)
	if len(missingRefs) == 0 && len(missingChallenges) == 0 && len(misclassified) == 0 && !gateMismatch &&
		len(counterfactualMismatches) == 0 && len(forbiddenClaimViolations) == 0 {
		result.Status = StatusPassed
		result.Passed = true
		result.NextAction = ""
	}
	return result
}

func failedResult(policy Policy, missingRefs, missingChallenges, misclassified []string, gateMismatch bool, counterfactualMismatches, forbiddenClaimViolations []string) Result {
	criticalFailures := len(missingRefs) + len(missingChallenges) + len(misclassified) +
		len(counterfactualMismatches) + len(forbiddenClaimViolations)
	if gateMismatch {
		criticalFailures++
	}
	return Result{
		Status:                   StatusFailed,
		Passed:                   false,
		MissingRefs:              missingRefs,
		MissingChallenges:        missingChallenges,
		MisclassifiedRefs:        misclassified,
		GateMismatch:             gateMismatch,
		CounterfactualMismatches: counterfactualMismatches,
		ForbiddenClaimViolations: forbiddenClaimViolations,
		CriticalFailures:         criticalFailures,
		NextAction:               policy.OnFailure,
	}
}

func missingRequiredTypes(required []string, challenges []Challenge) []string {
	present := map[string]bool{}
	for _, ch := range challenges {
		present[ch.Type] = true
	}
	var missing []string
	for _, typ := range required {
		if !present[typ] {
			missing = append(missing, typ)
		}
	}
	return missing
}

func missingSourceRefs(challenges []Challenge, understood []string) []string {
	seen := toRefSet(understood)
	missingSet := map[string]bool{}
	var missing []string
	for _, ch := range challenges {
		missing = appendUnseenRefs(missing, ch.SourceRefs, seen, missingSet)
	}
	return missing
}

func toRefSet(refs []string) map[string]bool {
	set := make(map[string]bool, len(refs))
	for _, ref := range refs {
		set[ref] = true
	}
	return set
}

// appendUnseenRefs appends each ref not already in seen or missingSet,
// recording it in missingSet so a ref repeated across challenges is only
// added to missing once.
func appendUnseenRefs(missing, refs []string, seen, missingSet map[string]bool) []string {
	for _, ref := range refs {
		if seen[ref] || missingSet[ref] {
			continue
		}
		missing = append(missing, ref)
		missingSet[ref] = true
	}
	return missing
}

func misclassifiedRefs(challenges []Challenge, got map[string]string) []string {
	var bad []string
	for _, ch := range challenges {
		for ref, expected := range ch.ExpectedClassification {
			if got[ref] != expected {
				bad = append(bad, ref)
			}
		}
	}
	return bad
}

func gateMismatch(challenges []Challenge, got *bool) bool {
	for _, ch := range challenges {
		if ch.ExpectedGateAllowed == nil {
			continue
		}
		if got == nil || *got != *ch.ExpectedGateAllowed {
			return true
		}
	}
	return false
}

// counterfactualMismatches reports the IDs of counterfactual challenges
// whose given answer is missing or does not match ExpectedCounterfactual.
func counterfactualMismatches(challenges []Challenge, got map[string]bool) []string {
	var mismatches []string
	for _, ch := range challenges {
		if ch.Type != ChallengeCounterfactual || ch.ExpectedCounterfactual == nil {
			continue
		}
		answer, ok := got[ch.ID]
		if !ok || answer != *ch.ExpectedCounterfactual {
			mismatches = append(mismatches, ch.ID)
		}
	}
	return mismatches
}

// forbiddenClaimViolations checks ack against policy-level ForbiddenClaims —
// a safety net independent of which challenges were generated. Returns the
// violated claim strings, in the order declared in claims.
func forbiddenClaimViolations(claims []string, ack Acknowledgment) []string {
	var violations []string
	for _, claim := range claims {
		if claim == ForbiddenClaimExecutionAuthorized {
			if ack.GateAllowed != nil && *ack.GateAllowed {
				violations = append(violations, claim)
			}
			continue
		}
		if ref, ok := strings.CutSuffix(claim, forbiddenClaimAsApprovedSuffix); ok {
			if ack.Classifications[ref] == DecisionApproved {
				violations = append(violations, claim)
			}
		}
	}
	return violations
}

package handoff

// Verify evaluates Sniper acknowledgment against Archivist challenges.
func Verify(policy Policy, challenges []Challenge, ack Acknowledgment) Result {
	if err := ValidatePolicy(policy); err != nil {
		return failedResult(policy, []string{err.Error()}, nil, nil, false)
	}
	if !policy.Enabled {
		return Result{Status: StatusSkipped, Passed: true}
	}

	missingChallenges := missingRequiredTypes(policy.RequiredTypes, challenges)
	missingRefs := missingSourceRefs(challenges, ack.UnderstoodRefs)
	misclassified := misclassifiedRefs(challenges, ack.Classifications)
	gateMismatch := gateMismatch(challenges, ack.GateAllowed)

	result := failedResult(policy, missingRefs, missingChallenges, misclassified, gateMismatch)
	if len(missingRefs) == 0 && len(missingChallenges) == 0 && len(misclassified) == 0 && !gateMismatch {
		result.Status = StatusPassed
		result.Passed = true
		result.NextAction = ""
	}
	return result
}

func failedResult(policy Policy, missingRefs, missingChallenges, misclassified []string, gateMismatch bool) Result {
	criticalFailures := len(missingRefs) + len(missingChallenges) + len(misclassified)
	if gateMismatch {
		criticalFailures++
	}
	return Result{
		Status:            StatusFailed,
		Passed:            false,
		MissingRefs:       missingRefs,
		MissingChallenges: missingChallenges,
		MisclassifiedRefs: misclassified,
		GateMismatch:      gateMismatch,
		CriticalFailures:  criticalFailures,
		NextAction:        policy.OnFailure,
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
	seen := map[string]bool{}
	for _, ref := range understood {
		seen[ref] = true
	}
	missingSet := map[string]bool{}
	var missing []string
	for _, ch := range challenges {
		for _, ref := range ch.SourceRefs {
			if seen[ref] || missingSet[ref] {
				continue
			}
			missing = append(missing, ref)
			missingSet[ref] = true
		}
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

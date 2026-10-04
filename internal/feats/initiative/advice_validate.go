package initiative

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Validate checks that the advice's identity, trigger, recommendation, and
// diligence fields are structurally complete.
func (a Advice) Validate() error {
	if !a.hasIdentity() {
		return fmt.Errorf("initiative_advice_invalid: identity fields are required")
	}
	if !a.hasValidTriggerPair() {
		return fmt.Errorf("initiative_advice_invalid: invalid trigger/supersedes pair")
	}
	if !a.hasValidRecommendation() {
		return fmt.Errorf("initiative_advice_invalid: recommendation is incomplete")
	}
	if !a.hasValidDiligence() {
		return fmt.Errorf("initiative_advice_invalid: diligence or alignment is incomplete")
	}
	if a.Leveling != nil {
		if err := a.Leveling.ValidateFor(a.Role); err != nil {
			return err
		}
	}
	return nil
}

func (a Advice) hasIdentity() bool {
	return a.AdviceID != "" && a.MissionID != "" && a.Role != "" &&
		a.RunID != "" && a.PolicyVersion != "" && a.PolicyDigest != ""
}

func (a Advice) hasValidTriggerPair() bool {
	if !validTriggers[a.Trigger] {
		return false
	}
	return a.Trigger == TriggerInitial || a.Supersedes != ""
}

func (a Advice) hasValidRecommendation() bool {
	return validEffort(a.Recommendation.RecommendedEffort) && a.Recommendation.RecommendedCapability != ""
}

func (a Advice) hasValidDiligence() bool {
	return validAlignment(a.Alignment) && len(a.Diligence.Checks) > 0 &&
		validateConfidenceTier(a.Diligence.ConfidenceCeiling) == nil &&
		uniqueNonEmpty(a.Diligence.Checks)
}

// uniqueNonEmpty reports whether every check is non-blank and appears once.
func uniqueNonEmpty(checks []string) bool {
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if strings.TrimSpace(check) == "" {
			return false
		}
		if _, exists := seen[check]; exists {
			return false
		}
		seen[check] = struct{}{}
	}
	return true
}

// ValidateAdviceJSON decodes raw as an Advice envelope, rejecting any
// top-level fields that belong to LEVELING, and validates the result.
func ValidateAdviceJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("initiative_advice_invalid: decode: %w", err)
	}
	for _, forbidden := range []string{"model", "provider", "effort", "capability", "level", "level_source", "resolved_execution_level"} {
		if _, ok := fields[forbidden]; ok {
			return fmt.Errorf("initiative_authority_violation: top-level %q belongs to LEVELING", forbidden)
		}
	}
	var advice Advice
	if err := json.Unmarshal(raw, &advice); err != nil {
		return fmt.Errorf("initiative_advice_invalid: decode envelope: %w", err)
	}
	return advice.Validate()
}

// AlignmentFor compares an observed effort level against the recommended
// one and returns the resulting AlignmentState.
func AlignmentFor(observed Observation, recommended EffortTier) AlignmentState {
	if observed.State == ObservationUnavailable {
		return AlignmentUnavailable
	}
	if observed.State == ObservationUnknown || observed.Effort == "" {
		return AlignmentUnknown
	}
	observedRank, observedOK := effortRank(observed.Effort)
	recommendedRank, recommendedOK := effortRank(recommended)
	if !observedOK || !recommendedOK {
		return AlignmentNotComparable
	}
	switch {
	case observedRank < recommendedRank:
		return AlignmentBelowRecommendation
	case observedRank > recommendedRank:
		return AlignmentAboveRecommendation
	default:
		return AlignmentMatched
	}
}

func normalizeObservation(observation Observation) Observation {
	if observation.State == "" {
		if observation.Effort == "" && observation.Model == "" && observation.Provider == "" {
			observation.State = ObservationUnavailable
		} else {
			observation.State = ObservationKnown
		}
	}
	return observation
}

func validAlignment(state AlignmentState) bool {
	switch state {
	case AlignmentMatched, AlignmentBelowRecommendation, AlignmentAboveRecommendation, AlignmentUnknown, AlignmentUnavailable, AlignmentNotComparable:
		return true
	default:
		return false
	}
}

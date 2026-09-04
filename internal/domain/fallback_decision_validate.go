package domain

import "fmt"

// FallbackDecisionFacts is the minimal set of facts a recorded provider-fallback
// degradation event (ADR-0028) carries that DecideSlotFallbackOutcome needs to
// recompute the outcome independently. FallbackAvailable is always true for a
// recorded decision — a decision is only ever recorded when a fallback was
// actually applied (see telemetry.FallbackDecision), so there is nothing to
// reconstruct it from beyond the record itself.
type FallbackDecisionFacts struct {
	Slot          string
	Policy        ResolutionPolicy
	Outcome       FallbackOutcome
	UserConfirmed bool
}

// ValidateFallbackDecision recomputes DecideSlotFallbackOutcome from facts and
// reports an error if the claimed Outcome does not match what the policy table
// prescribes, or if an ask_required outcome is claimed without UserConfirmed —
// the two ways a fallback record could misrepresent the Strategist Approval
// Gate having been honored. This is the closed-loop check for fallback
// decisions, mirroring how ValidateRouteDecision checks Scout's route
// decisions against request facts.
func ValidateFallbackDecision(facts FallbackDecisionFacts) error {
	want := DecideSlotFallbackOutcome(facts.Slot, facts.Policy, true)
	if facts.Outcome != want {
		return fmt.Errorf(
			"fallback decision outcome %q does not match policy table result %q for slot=%s policy=%s",
			facts.Outcome, want, facts.Slot, facts.Policy.EffectivePolicy(),
		)
	}
	if facts.Outcome == FallbackOutcomeAskRequired && !facts.UserConfirmed {
		return fmt.Errorf("fallback decision outcome=%s requires explicit user confirmation before being recorded as applied", FallbackOutcomeAskRequired)
	}
	return nil
}

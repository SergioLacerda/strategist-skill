package domain

import (
	"strings"
	"testing"
)

func TestValidateFallbackDecision_MatchesPolicyTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		facts FallbackDecisionFacts
	}{
		{
			name: "native policy auto_native",
			facts: FallbackDecisionFacts{
				Slot:    "execution",
				Policy:  ResolutionPolicyNative,
				Outcome: FallbackOutcomeAutoNative,
			},
		},
		{
			name: "ask policy ask_required with confirmation",
			facts: FallbackDecisionFacts{
				Slot:          "refinement",
				Policy:        ResolutionPolicyAsk,
				Outcome:       FallbackOutcomeAskRequired,
				UserConfirmed: true,
			},
		},
		{
			name: "empty policy defaults to ask, ask_required with confirmation",
			facts: FallbackDecisionFacts{
				Slot:          "execution",
				Policy:        "",
				Outcome:       FallbackOutcomeAskRequired,
				UserConfirmed: true,
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateFallbackDecision(tc.facts); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateFallbackDecision_RejectsOutcomeMismatch(t *testing.T) {
	t.Parallel()
	facts := FallbackDecisionFacts{
		Slot:    "execution",
		Policy:  ResolutionPolicyBlock,
		Outcome: FallbackOutcomeAutoNative,
	}
	err := ValidateFallbackDecision(facts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match policy table result") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFallbackDecision_RejectsUnconfirmedAsk(t *testing.T) {
	t.Parallel()
	facts := FallbackDecisionFacts{
		Slot:          "execution",
		Policy:        ResolutionPolicyAsk,
		Outcome:       FallbackOutcomeAskRequired,
		UserConfirmed: false,
	}
	err := ValidateFallbackDecision(facts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "requires explicit user confirmation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFallbackDecision_DiscoverySlotAlwaysNativeMismatchesAskOrAutoNative(t *testing.T) {
	t.Parallel()
	// Discovery is exempt from provider_resolution_policy entirely
	// (FallbackOutcomeAlwaysNative) — a recorded decision claiming
	// ask_required or auto_native for discovery can never match the table.
	facts := FallbackDecisionFacts{
		Slot:    "discovery",
		Policy:  ResolutionPolicyNative,
		Outcome: FallbackOutcomeAutoNative,
	}
	err := ValidateFallbackDecision(facts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "always_native_no_policy") {
		t.Fatalf("expected error to mention always_native_no_policy outcome, got: %v", err)
	}
}

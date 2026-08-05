package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredByRisk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   RiskSignals
		want bool
	}{
		{name: "skip informational only", in: RiskSignals{InformationalOnly: true}, want: false},
		{name: "approval gate requires challenge", in: RiskSignals{ApprovalGatePresent: true}, want: true},
		{name: "mandatory constraints require challenge", in: RiskSignals{MandatoryConstraintsPresent: true}, want: true},
		{name: "unresolved questions require challenge", in: RiskSignals{UnresolvedQuestionsPresent: true}, want: true},
		{name: "forbidden scope requires challenge", in: RiskSignals{ForbiddenScopePresent: true}, want: true},
		{name: "implementation handoff requires challenge", in: RiskSignals{ImplementationHandoffPresent: true}, want: true},
		{name: "destructive operation requires challenge", in: RiskSignals{DestructiveOperationPossible: true}, want: true},
		{name: "security task requires challenge", in: RiskSignals{SecuritySensitiveTask: true}, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, RequiredByRisk(tt.in))
		})
	}
}

func TestValidatePolicy(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidatePolicy(DefaultPolicy()))
	require.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: TransitionArchivistToSniper}), "required_types")
	require.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: "some_other_transition", RequiredTypes: []string{ChallengeObjective}}), "not supported")
	require.ErrorContains(t, ValidatePolicy(Policy{Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{"essay"}}), "not allowed")
}

func TestValidatePolicy_RangerToArchivistTransition(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidatePolicy(Policy{
		Enabled:       true,
		Transition:    TransitionRangerToArchivist,
		RequiredTypes: []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict},
		MaxAttempts:   2,
	}))
}

func TestValidatePolicy_ChallengeTypesAreTransitionScoped(t *testing.T) {
	t.Parallel()

	// MVP-only types are rejected on the Ranger->Archivist transition.
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionRangerToArchivist, RequiredTypes: []string{ChallengeGate},
	}), "not allowed for transition")
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionRangerToArchivist, RequiredTypes: []string{ChallengeObjective},
	}), "not allowed for transition")

	// Ranger->Archivist-only types are rejected on the MVP transition.
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{ChallengeRecall},
	}), "not allowed for transition")
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{ChallengeVerdict},
	}), "not allowed for transition")

	// boundary/classification are shared vocabulary, valid on both.
	assert.NoError(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{ChallengeBoundary, ChallengeClassification},
	}))
	assert.NoError(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionRangerToArchivist, RequiredTypes: []string{ChallengeBoundary, ChallengeClassification},
	}))
}

func TestValidatePolicy_SniperToValidationTransition(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidatePolicy(Policy{
		Enabled:       true,
		Transition:    TransitionSniperToValidation,
		RequiredTypes: []string{ChallengeBoundary, ChallengeClassification, ChallengeCounterfactual},
		MaxAttempts:   1,
	}))

	// Ranger->Archivist-only types are rejected on the new transition.
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionSniperToValidation, RequiredTypes: []string{ChallengeRecall},
	}), "not allowed for transition")
	// MVP-only types (objective, gate) are rejected on the new transition too.
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionSniperToValidation, RequiredTypes: []string{ChallengeGate},
	}), "not allowed for transition")
}

func TestValidatePolicy_CounterfactualAllowedOnMVPAndValidationOnly(t *testing.T) {
	t.Parallel()

	assert.NoError(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionArchivistToSniper, RequiredTypes: []string{ChallengeCounterfactual},
	}))
	assert.NoError(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionSniperToValidation, RequiredTypes: []string{ChallengeCounterfactual},
	}))
	require.ErrorContains(t, ValidatePolicy(Policy{
		Enabled: true, Transition: TransitionRangerToArchivist, RequiredTypes: []string{ChallengeCounterfactual},
	}), "not allowed for transition")
}

func TestValidatePolicy_ForbiddenClaims(t *testing.T) {
	t.Parallel()

	valid := DefaultPolicy()
	valid.ForbiddenClaims = []string{ForbiddenClaimExecutionAuthorized, "Q-001_as_approved"}
	require.NoError(t, ValidatePolicy(valid))

	invalid := DefaultPolicy()
	invalid.ForbiddenClaims = []string{"Q-001_is_approved"} // wrong suffix
	require.ErrorContains(t, ValidatePolicy(invalid), "forbidden_claims entry")

	emptyRef := DefaultPolicy()
	emptyRef.ForbiddenClaims = []string{forbiddenClaimAsApprovedSuffix} // no ref before the suffix
	require.ErrorContains(t, ValidatePolicy(emptyRef), "forbidden_claims entry")
}

func TestDefaultPolicy_UnchangedByRangerToArchivistAddition(t *testing.T) {
	t.Parallel()

	// Acceptance check: the MVP's archivist_to_sniper behavior is additive-only.
	p := DefaultPolicy()
	assert.Equal(t, TransitionArchivistToSniper, p.Transition)
	assert.Equal(t, []string{ChallengeObjective, ChallengeBoundary, ChallengeClassification, ChallengeGate}, p.RequiredTypes)
	require.NoError(t, ValidatePolicy(p))
}

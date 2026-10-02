package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRangerToArchivistPolicy(t *testing.T) {
	t.Parallel()

	p := RangerToArchivistPolicy()
	assert.False(t, p.Enabled, "advisory-first: disabled by default")
	assert.Equal(t, TransitionRangerToArchivist, p.Transition)
	assert.ElementsMatch(t, []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict}, p.RequiredTypes)
	assert.True(t, p.RequireAllCritical)
	assert.Equal(t, 2, p.MaxAttempts)
	assert.Equal(t, FailureActionReturnToArchivist, p.OnFailure)
}

func TestSniperToValidationPolicy(t *testing.T) {
	t.Parallel()

	p := SniperToValidationPolicy()
	assert.False(t, p.Enabled, "advisory-first: disabled by default")
	assert.Equal(t, TransitionSniperToValidation, p.Transition)
	assert.ElementsMatch(t, []string{ChallengeBoundary, ChallengeClassification}, p.RequiredTypes)
	assert.True(t, p.RequireAllCritical)
	assert.Equal(t, 2, p.MaxAttempts)
	assert.Equal(t, FailureActionReturnToArchivist, p.OnFailure)
}

func TestStatusForRisk(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StatusRequired, StatusForRisk(RiskSignals{ApprovalGatePresent: true}))
	assert.Equal(t, StatusSkipped, StatusForRisk(RiskSignals{InformationalOnly: true}))
}

func TestRiskSignalsForLevel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StatusRequired, StatusForRisk(RiskSignalsForLevel("high")), "high risk_level must require a challenge")
	assert.Equal(t, StatusRequired, StatusForRisk(RiskSignalsForLevel("medium")), "medium risk_level must require a challenge")
	assert.Equal(t, StatusSkipped, StatusForRisk(RiskSignalsForLevel("low")), "low risk_level must not require a challenge")
	assert.Equal(t, StatusSkipped, StatusForRisk(RiskSignalsForLevel("")), "unknown risk_level defaults to advisory (skipped), not fail-closed")
	assert.Equal(t, StatusSkipped, StatusForRisk(RiskSignalsForLevel("not_a_real_level")), "unrecognized risk_level defaults to advisory (skipped)")
}

func TestResolvePolicyForMission_HighRiskEnablesRangerToArchivist(t *testing.T) {
	t.Parallel()

	p, err := ResolvePolicyForMission("high", TransitionRangerToArchivist)
	require.NoError(t, err)
	assert.True(t, p.Enabled, "high-risk mission must enable the Ranger->Archivist challenge")
	assert.Equal(t, TransitionRangerToArchivist, p.Transition)
	assert.ElementsMatch(t, []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict}, p.RequiredTypes,
		"enabled policy keeps the transition's risk-appropriate required challenge types")
}

func TestResolvePolicyForMission_LowRiskLeavesRangerToArchivistDisabled(t *testing.T) {
	t.Parallel()

	p, err := ResolvePolicyForMission("low", TransitionRangerToArchivist)
	require.NoError(t, err)
	assert.False(t, p.Enabled, "low-risk mission must not require the Ranger->Archivist challenge")
	assert.Empty(t, p.RequiredTypes, "a disabled policy carries no required types to enforce")
}

func TestResolvePolicyForMission_HighRiskEnablesSniperToValidation(t *testing.T) {
	t.Parallel()

	p, err := ResolvePolicyForMission("high", TransitionSniperToValidation)
	require.NoError(t, err)
	assert.True(t, p.Enabled)
	assert.Equal(t, TransitionSniperToValidation, p.Transition)
	assert.ElementsMatch(t, []string{ChallengeBoundary, ChallengeClassification}, p.RequiredTypes)
}

func TestResolvePolicyForMission_LowRiskLeavesSniperToValidationDisabled(t *testing.T) {
	t.Parallel()

	p, err := ResolvePolicyForMission("low", TransitionSniperToValidation)
	require.NoError(t, err)
	assert.False(t, p.Enabled)
	assert.Empty(t, p.RequiredTypes)
}

func TestResolvePolicyForMission_ArchivistToSniperIsNotResolvedFromACoarseLabel(t *testing.T) {
	t.Parallel()

	for _, level := range []string{"low", "medium", "high", ""} {
		_, err := ResolvePolicyForMission(level, TransitionArchivistToSniper)
		require.ErrorContains(t, err, "ExtractRiskSignals", level)
	}
}

func TestResolveArchivistPolicy_SkipsOnlyAFullyInformationalPackage(t *testing.T) {
	t.Parallel()

	skipped := ResolveArchivistPolicy(RiskSignals{InformationalOnly: true})
	assert.False(t, skipped.Enabled)
	assert.Empty(t, skipped.RequiredTypes)
	assert.True(t, skipped.RequirePrecedence)

	for name, signals := range map[string]RiskSignals{
		"nothing declared":        {},
		"implementation handoff":  {InformationalOnly: true, ImplementationHandoffPresent: true},
		"mandatory constraints":   {InformationalOnly: true, MandatoryConstraintsPresent: true},
		"unresolved questions":    {InformationalOnly: true, UnresolvedQuestionsPresent: true},
		"forbidden scope":         {InformationalOnly: true, ForbiddenScopePresent: true},
		"destructive operation":   {InformationalOnly: true, DestructiveOperationPossible: true},
		"security sensitive task": {InformationalOnly: true, SecuritySensitiveTask: true},
	} {
		policy := ResolveArchivistPolicy(signals)
		assert.True(t, policy.Enabled, name)
		assert.NotEmpty(t, policy.RequiredTypes, name)
	}
}

func TestResolveArchivistPolicy_ApprovalGateIsNeverAnInput(t *testing.T) {
	t.Parallel()

	assert.False(t, ResolveArchivistPolicy(RiskSignals{InformationalOnly: true, ApprovalGatePresent: true}).Enabled,
		"the Approval Gate is independent: its presence neither requires nor skips the challenge")
}

func TestPolicyIdentityIsStableAndIgnoresPerPackageEnablement(t *testing.T) {
	t.Parallel()

	required := ResolveArchivistPolicy(RiskSignals{})
	skipped := ResolveArchivistPolicy(RiskSignals{InformationalOnly: true})
	assert.Equal(t, PolicyIdentity(required), PolicyIdentity(skipped))
	changed := DefaultPolicy()
	changed.MaxAttempts++
	assert.NotEqual(t, PolicyIdentity(DefaultPolicy()), PolicyIdentity(changed))
}

func TestResolvePolicyForMission_UnknownTransitionErrors(t *testing.T) {
	t.Parallel()

	_, err := ResolvePolicyForMission("high", "not_a_real_transition")
	require.ErrorContains(t, err, "unknown transition")
}

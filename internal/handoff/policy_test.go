package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyPassesCompleteAcknowledgment(t *testing.T) {
	t.Parallel()

	gateAllowed := false
	result := Verify(DefaultPolicy(), sampleChallenges(gateAllowed), Acknowledgment{
		ChallengeRefs:  []string{"HC-001", "HC-002", "HC-003", "HC-004"},
		UnderstoodRefs: []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
		Classifications: map[string]string{
			"D-001": DecisionApproved,
			"Q-001": QuestionUnresolved,
		},
		GateAllowed: &gateAllowed,
	})

	require.True(t, result.Passed)
	assert.Equal(t, StatusPassed, result.Status)
	assert.Zero(t, result.CriticalFailures)
}

func TestVerifyFailsMissingAcknowledgment(t *testing.T) {
	t.Parallel()

	gateAllowed := false
	result := Verify(DefaultPolicy(), sampleChallenges(gateAllowed), Acknowledgment{
		UnderstoodRefs: []string{"G-001"},
	})

	require.False(t, result.Passed)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Contains(t, result.MissingRefs, "X-001")
	assert.Contains(t, result.MissingRefs, "D-001")
	assert.Contains(t, result.MissingRefs, "Q-001")
	assert.True(t, result.GateMismatch)
	assert.Equal(t, FailureActionReturnToArchivist, result.NextAction)
}

func TestVerifyFailsMisclassifiedOpenQuestion(t *testing.T) {
	t.Parallel()

	gateAllowed := false
	result := Verify(DefaultPolicy(), sampleChallenges(gateAllowed), Acknowledgment{
		UnderstoodRefs: []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
		Classifications: map[string]string{
			"D-001": DecisionApproved,
			"Q-001": DecisionApproved,
		},
		GateAllowed: &gateAllowed,
	})

	require.False(t, result.Passed)
	assert.Contains(t, result.MisclassifiedRefs, "Q-001")
}

func TestVerifySkipsDisabledPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultPolicy()
	policy.Enabled = false

	result := Verify(policy, nil, Acknowledgment{})

	require.True(t, result.Passed)
	assert.Equal(t, StatusSkipped, result.Status)
}

func sampleChallenges(gateAllowed bool) []Challenge {
	return []Challenge{
		{ID: "HC-001", Type: ChallengeObjective, SourceRefs: []string{"G-001"}, Critical: true},
		{ID: "HC-002", Type: ChallengeBoundary, SourceRefs: []string{"X-001"}, Critical: true},
		{
			ID:         "HC-003",
			Type:       ChallengeClassification,
			SourceRefs: []string{"D-001", "Q-001"},
			Critical:   true,
			ExpectedClassification: map[string]string{
				"D-001": DecisionApproved,
				"Q-001": QuestionUnresolved,
			},
		},
		{ID: "HC-004", Type: ChallengeGate, SourceRefs: []string{"approval.required"}, Critical: true, ExpectedGateAllowed: &gateAllowed},
	}
}

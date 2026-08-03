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

func rangerToArchivistPolicy() Policy {
	return Policy{
		Enabled:            true,
		Transition:         TransitionRangerToArchivist,
		RequiredTypes:      []string{ChallengeRecall, ChallengeBoundary, ChallengeClassification, ChallengeVerdict},
		RequireAllCritical: true,
		MaxAttempts:        2,
		OnFailure:          FailureActionReturnToArchivist,
	}
}

// verdictChallenges mirrors sampleChallenges' shape but for the
// Ranger->Archivist transition: recall/boundary/classification/verdict
// instead of objective/boundary/classification/gate. The verdict challenge
// reuses the generic ExpectedClassification mechanism — no new Verify
// mechanics were needed for the new transition, per design.md § Item 1.
func verdictChallenges() []Challenge {
	return []Challenge{
		{ID: "HC-101", Type: ChallengeRecall, SourceRefs: []string{"KF-001"}, Critical: true},
		{ID: "HC-102", Type: ChallengeBoundary, SourceRefs: []string{"SQ-001"}, Critical: true},
		{ID: "HC-103", Type: ChallengeClassification, SourceRefs: []string{"KF-001", "UN-001"}, Critical: true,
			ExpectedClassification: map[string]string{"KF-001": "known_fact", "UN-001": "uncertainty"}},
		{ID: "HC-104", Type: ChallengeVerdict, SourceRefs: []string{"VERDICT"}, Critical: true,
			ExpectedClassification: map[string]string{"VERDICT": "partially_implemented"}},
	}
}

func TestVerify_VerdictChallengePasses(t *testing.T) {
	t.Parallel()

	result := Verify(rangerToArchivistPolicy(), verdictChallenges(), Acknowledgment{
		UnderstoodRefs: []string{"KF-001", "SQ-001", "UN-001", "VERDICT"},
		Classifications: map[string]string{
			"KF-001":  "known_fact",
			"UN-001":  "uncertainty",
			"VERDICT": "partially_implemented",
		},
	})

	require.True(t, result.Passed)
	assert.Equal(t, StatusPassed, result.Status)
	assert.Zero(t, result.CriticalFailures)
}

func TestVerify_VerdictChallengeFailsOnMisrestatedVerdict(t *testing.T) {
	t.Parallel()

	result := Verify(rangerToArchivistPolicy(), verdictChallenges(), Acknowledgment{
		UnderstoodRefs: []string{"KF-001", "SQ-001", "UN-001", "VERDICT"},
		Classifications: map[string]string{
			"KF-001":  "known_fact",
			"UN-001":  "uncertainty",
			"VERDICT": "implemented", // wrong — the true verdict is partially_implemented
		},
	})

	require.False(t, result.Passed)
	assert.Equal(t, StatusFailed, result.Status)
	assert.Contains(t, result.MisclassifiedRefs, "VERDICT")
	assert.Equal(t, FailureActionReturnToArchivist, result.NextAction)
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

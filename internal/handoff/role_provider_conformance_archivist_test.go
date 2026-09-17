package handoff_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/assert"
)

// TestHandoffRequiredChallengeTypesMatchArchivistToSniperContract proves the
// canonical Archivist->Sniper handoff policy still requires every challenge
// type it declares. This file is separate from the Ranger conformance suite so
// Ranked certification can pin independent evidence for each role.
func TestHandoffRequiredChallengeTypesMatchArchivistToSniperContract(t *testing.T) {
	t.Parallel()

	policy := handoff.DefaultPolicy()

	gateAllowed := true
	challenges := []handoff.Challenge{
		{ID: "C-01", Type: handoff.ChallengeObjective, SourceRefs: []string{"KF-01"}},
		{ID: "C-02", Type: handoff.ChallengeBoundary, SourceRefs: []string{"KF-01"}},
		{ID: "C-03", Type: handoff.ChallengeClassification, ExpectedClassification: map[string]string{"KF-01": handoff.DecisionApproved}},
		{ID: "C-04", Type: handoff.ChallengeGate, ExpectedGateAllowed: &gateAllowed},
	}
	ack := handoff.Acknowledgment{
		UnderstoodRefs:  []string{"KF-01"},
		Classifications: map[string]string{"KF-01": handoff.DecisionApproved},
		GateAllowed:     &gateAllowed,
	}

	result := handoff.Verify(policy, challenges, ack)
	assert.True(t, result.Passed)

	for i := range challenges {
		withoutOne := append([]handoff.Challenge(nil), challenges[:i]...)
		withoutOne = append(withoutOne, challenges[i+1:]...)
		result := handoff.Verify(policy, withoutOne, ack)
		assert.Falsef(t, result.Passed, "dropping challenge type %s must fail verification", challenges[i].Type)
		assert.Contains(t, result.MissingChallenges, challenges[i].Type)
	}
}

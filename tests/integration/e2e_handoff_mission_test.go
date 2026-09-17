//go:build integration

package integration_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	mission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestE2E_LiveArchivistToSniperChallengeEnforcesTransition(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "e2e-live-handoff"})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone,
		domain.MissionEventIntakeDone,
		domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone,
		domain.MissionEventGateApproved,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err)
	}

	failedStatus, failed, err := mission.ArchivistToSniper(engine, root, mission.LiveHandoffInput{
		Policy:     handoff.DefaultPolicy(),
		Challenges: e2eChallenges(false),
		Ack:        handoff.Acknowledgment{},
		Attempt:    1,
	})
	require.NoError(t, err)
	require.False(t, failed.Passed)
	require.Equal(t, domain.StateRefinement, failedStatus.State)

	_, err = engine.Submit(domain.MissionEventRefinementDone)
	require.NoError(t, err)
	_, err = engine.Submit(domain.MissionEventGateApproved)
	require.NoError(t, err)
	passedStatus, passed, err := mission.ArchivistToSniper(engine, root, mission.LiveHandoffInput{
		Policy:     handoff.DefaultPolicy(),
		Challenges: e2eChallenges(false),
		Ack:        e2eValidAck(),
		Attempt:    2,
	})
	require.NoError(t, err)
	require.True(t, passed.Passed)
	require.Equal(t, domain.StateExecution, passedStatus.State)

	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.False(t, records[0].Passed)
	require.True(t, records[1].Passed)
}

func e2eChallenges(gateAllowed bool) []handoff.Challenge {
	return []handoff.Challenge{
		{ID: "HC-001", Type: handoff.ChallengeObjective, SourceRefs: []string{"G-001"}, Critical: true},
		{ID: "HC-002", Type: handoff.ChallengeBoundary, SourceRefs: []string{"X-001"}, Critical: true},
		{ID: "HC-003", Type: handoff.ChallengeClassification, SourceRefs: []string{"D-001", "Q-001"}, Critical: true,
			ExpectedClassification: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved}},
		{ID: "HC-004", Type: handoff.ChallengeGate, SourceRefs: []string{"approval.required"}, Critical: true, ExpectedGateAllowed: &gateAllowed},
	}
}

func e2eValidAck() handoff.Acknowledgment {
	gateAllowed := false
	return handoff.Acknowledgment{
		ChallengeRefs:   []string{"HC-001", "HC-002", "HC-003", "HC-004"},
		UnderstoodRefs:  []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
		Classifications: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved},
		GateAllowed:     &gateAllowed,
	}
}

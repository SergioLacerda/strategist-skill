package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestArchivistToSniper_PersistsBeforePermittingExecution(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine := readyForHandoff(t, "live-pass")
	allowed := false
	resultChallenges := liveChallenges(allowed)
	status, result, err := ArchivistToSniper(engine, root, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: resultChallenges,
		Ack: validAck(), Attempt: 1,
	})
	require.NoError(t, err)
	require.True(t, result.Passed)
	require.Equal(t, domain.StateExecution, status.State)
	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "live-pass", records[0].MissionID)
	require.True(t, records[0].Passed)
	confidenceRecords, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, confidenceRecords, 1)
	require.Equal(t, telemetry.ConfidenceCoverageMissing, confidenceRecords[0].CoverageStatus)
	require.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, confidenceRecords[0].Agent)
}

func TestArchivistToSniper_FailureReturnsAndExhaustsWithoutExecution(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine := readyForHandoff(t, "live-fail")
	status, result, err := ArchivistToSniper(engine, root, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: liveChallenges(false),
		Ack: handoff.Acknowledgment{}, Attempt: 1,
	})
	require.NoError(t, err)
	require.False(t, result.Passed)
	require.Equal(t, domain.StateRefinement, status.State)
	require.Equal(t, 1, status.HandoffAttempt)

	status, _, err = ArchivistToSniper(readyForHandoffAgain(t, engine), root, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: liveChallenges(false),
		Ack: handoff.Acknowledgment{}, Attempt: 2,
	})
	require.NoError(t, err)
	require.Equal(t, domain.StateBlocked, status.State)
	require.NotEqual(t, domain.StateExecution, status.State)
	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
}

func TestArchivistToSniper_PersistenceFailureDoesNotAdvance(t *testing.T) {
	engine := readyForHandoff(t, "persist-fail")
	blocker := t.TempDir()
	path := filepath.Join(blocker, "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("block"), 0o600))
	status, _, err := ArchivistToSniper(engine, path, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: liveChallenges(false),
		Ack: validAck(), Attempt: 1,
	})
	require.Error(t, err)
	require.Equal(t, domain.StateHandoffChallenge, status.State)
}

func TestArchivistToSniper_PersistsOpenQuestions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine := readyForHandoff(t, "open-question")
	status, result, err := ArchivistToSniper(engine, root, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: liveChallenges(false),
		Ack: validAck(), Attempt: 1,
		ConfidenceSummary: &domain.ConfidenceSummary{
			PolicyVersion:     domain.ConfidencePolicyVersion,
			CalibrationStatus: domain.CalibrationNoSample,
			OpenQuestions: []domain.ConfidenceClaim{{
				ID: "Q-001", Statement: "Is the handoff complete?", Agent: "archivist",
				CorrelationKey: "question-1", ClaimKind: domain.ClaimKindQuestion,
				ConfidencePercent: 40,
			}},
		},
	})
	require.NoError(t, err)
	require.True(t, result.Passed)
	require.Equal(t, domain.StateExecution, status.State)
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "Q-001", records[0].ClaimID)
	require.Equal(t, domain.ClaimKindQuestion, records[0].ClaimKind)
}

func TestArchivistToSniper_InitiativeChallengeReturnsToRefinement(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	engine := readyForHandoff(t, "initiative-challenge")
	runtime, err := NewDefaultInitiativeRuntime(root)
	require.NoError(t, err)
	advice, err := runtime.EnterRole(InitiativeRoleEntry{MissionID: "initiative-challenge", Role: "archivist", RunID: "run-1"})
	require.NoError(t, err)
	handoffEnvelope, err := runtime.CompleteRole(advice, initiative.Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []initiative.ObligationCheck{{ID: "mandatory", Status: initiative.CheckBlocked}},
	}, "sniper")
	require.NoError(t, err)
	require.True(t, handoffEnvelope.Assessment.Challenge)

	status, result, err := ArchivistToSniper(engine, root, LiveHandoffInput{
		Policy: handoff.DefaultPolicy(), Challenges: liveChallenges(false),
		Ack: validAck(), Attempt: 1, Initiative: &handoffEnvelope,
	})
	require.NoError(t, err)
	require.False(t, result.Passed)
	require.Equal(t, domain.StateRefinement, status.State)
	require.NotEqual(t, domain.StateExecution, status.State)
	events, readErr := os.ReadFile(telemetry.InitiativeEventHistoryPath(root))
	require.NoError(t, readErr)
	require.Contains(t, string(events), "strategist.initiative.handoff_consumed")
}

func readyForHandoff(t *testing.T, id string) *domain.MissionEngine {
	t.Helper()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: id})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone,
		domain.MissionEventDiscoveryDone, domain.MissionEventRefinementDone,
		domain.MissionEventGateApproved,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err)
	}
	return engine
}

func readyForHandoffAgain(t *testing.T, engine *domain.MissionEngine) *domain.MissionEngine {
	t.Helper()
	_, err := engine.Submit(domain.MissionEventRefinementDone)
	require.NoError(t, err)
	_, err = engine.Submit(domain.MissionEventGateApproved)
	require.NoError(t, err)
	return engine
}

func liveChallenges(gateAllowed bool) []handoff.Challenge {
	return []handoff.Challenge{
		{ID: "HC-001", Type: handoff.ChallengeObjective, SourceRefs: []string{"G-001"}, Critical: true},
		{ID: "HC-002", Type: handoff.ChallengeBoundary, SourceRefs: []string{"X-001"}, Critical: true},
		{ID: "HC-003", Type: handoff.ChallengeClassification, SourceRefs: []string{"D-001", "Q-001"}, Critical: true,
			ExpectedClassification: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved}},
		{ID: "HC-004", Type: handoff.ChallengeGate, SourceRefs: []string{"approval.required"}, Critical: true, ExpectedGateAllowed: &gateAllowed},
	}
}

func validAck() handoff.Acknowledgment {
	gateAllowed := false
	return handoff.Acknowledgment{
		ChallengeRefs:   []string{"HC-001", "HC-002", "HC-003", "HC-004"},
		UnderstoodRefs:  []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
		Classifications: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved},
		GateAllowed:     &gateAllowed,
	}
}

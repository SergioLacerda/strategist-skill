//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	mission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

// TestE2E_LiveArchivistToSniperChallengeEnforcesTransition drives the
// production composition (mission.RecordArchivistHandoff) through the failure
// loop: a failed challenge returns to refinement and needs a new Approval Gate,
// the repaired attempt passes, and only the consumed outcome enters execution.
func TestE2E_LiveArchivistToSniperChallengeEnforcesTransition(t *testing.T) {
	t.Parallel()
	root, basePath := t.TempDir(), t.TempDir()
	writeE2EPackage(t, basePath, "e2e-live-handoff")
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "e2e-live-handoff"})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err)
	}
	_, err = engine.Submit(domain.MissionEventGateApproved)
	require.NoError(t, err)
	digest, err := handoff.PackageDigest(filepath.Join(basePath, "refined", "e2e-live-handoff"))
	require.NoError(t, err)
	_, err = engine.RecordApprovalGatePackageDigest(digest)
	require.NoError(t, err)

	failed, failedStatus, _, err := mission.RecordArchivistHandoff(root, basePath, engine, mission.ArchivistHandoffInput{
		Challenges: e2eChallenges(false), Ack: handoff.Acknowledgment{},
	})
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomeFailed, failed.Outcome.Result)
	require.Equal(t, domain.StateRefinement, failedStatus.State)

	_, err = engine.Submit(domain.MissionEventRefinementDone)
	require.NoError(t, err)
	_, err = engine.Submit(domain.MissionEventGateApproved)
	require.NoError(t, err)
	digest, err = handoff.PackageDigest(filepath.Join(basePath, "refined", "e2e-live-handoff"))
	require.NoError(t, err)
	_, err = engine.RecordApprovalGatePackageDigest(digest)
	require.NoError(t, err)
	passed, passedStatus, _, err := mission.RecordArchivistHandoff(root, basePath, engine, mission.ArchivistHandoffInput{
		Challenges: e2eChallenges(false), Ack: e2eValidAck(),
	})
	require.NoError(t, err)
	require.Equal(t, handoff.OutcomePassed, passed.Outcome.Result)
	require.Equal(t, 2, passedStatus.HandoffAttempt)
	require.Equal(t, domain.StateHandoffChallenge, passedStatus.State, "a passed evaluation does not enter execution by itself")

	outcome, err := mission.AuthorizeHandoffExecution(root, basePath, passedStatus)
	require.NoError(t, err)
	require.NoError(t, mission.ConsumeHandoffOutcome(root, outcome))
	executing, err := engine.Submit(domain.MissionEventHandoffSatisfied)
	require.NoError(t, err)
	require.Equal(t, domain.StateExecution, executing.State)

	records, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.False(t, records[0].Passed)
	require.True(t, records[1].Passed)
}

func writeE2EPackage(t *testing.T, basePath, missionID string) {
	t.Helper()
	dir := filepath.Join(basePath, "refined", missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	facts := "handoff_policy_facts:\n  schema_version: strategist-handoff-policy-facts/v1\n  mandatory_constraints: []\n  unresolved_questions: [\"which store?\"]\n  forbidden_scope: []\n  destructive_operation_possible: false\n  security_sensitive_task: false\n  informational_only: false\n"
	files := map[string]string{
		"analysis.md": "---\nmission_id: " + missionID + "\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": "- [ ] 1.1 [analysis_artifact] record the evidence\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
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

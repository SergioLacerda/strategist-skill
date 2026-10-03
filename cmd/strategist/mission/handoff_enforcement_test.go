package mission_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const requiredFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: ["which store?"]
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: false
`

const implementationTasks = "- [ ] 1.1 [task_type: implementation_handoff] change the code\n"

func submitSatisfied(t *testing.T, root, id string) (string, error) {
	t.Helper()
	return runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(domain.MissionEventHandoffSatisfied))
}

func passingChallenge() livemission.ArchivistHandoffInput {
	no := false
	return livemission.ArchivistHandoffInput{
		Challenges: []handoff.Challenge{
			{ID: "HC-001", Type: handoff.ChallengeObjective, SourceRefs: []string{"G-001"}, Critical: true},
			{ID: "HC-002", Type: handoff.ChallengeBoundary, SourceRefs: []string{"X-001"}, Critical: true},
			{ID: "HC-003", Type: handoff.ChallengeClassification, SourceRefs: []string{"D-001", "Q-001"}, Critical: true,
				ExpectedClassification: map[string]string{"D-001": "approved_decision", "Q-001": "unresolved_question"}},
			{ID: "HC-004", Type: handoff.ChallengeGate, SourceRefs: []string{"approval.required"}, Critical: true, ExpectedGateAllowed: &no},
		},
		Ack: handoff.Acknowledgment{
			ChallengeRefs:   []string{"HC-001", "HC-002", "HC-003", "HC-004"},
			UnderstoodRefs:  []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
			Classifications: map[string]string{"D-001": "approved_decision", "Q-001": "unresolved_question"},
			GateAllowed:     &no,
		},
	}
}

func failingChallenge() livemission.ArchivistHandoffInput {
	input := passingChallenge()
	yes := true
	input.Ack.GateAllowed = &yes
	input.Ack.Classifications["Q-001"] = "approved_decision"
	return input
}

func handoffRoot(t *testing.T, id, facts, tasks string) string {
	t.Helper()
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallengeWithPackage(t, root, id, facts, tasks)
	return root
}

func outcomeFiles(t *testing.T, root, id string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "missions", "handoff", id))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestInformationalPackageSkipIsRecordedWithProvenanceAndAuthorizesEntry(t *testing.T) {
	root := handoffRoot(t, "m-skip", informationalFacts, analysisOnlyTasks)

	evaluation, err := evaluateHandoff(t, root, "m-skip", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)

	assert.Equal(t, handoff.OutcomeSkipped, evaluation.Outcome.Result)
	assert.False(t, evaluation.Outcome.Required)
	assert.True(t, evaluation.Outcome.Signals.InformationalOnly)
	assert.Equal(t, string(domain.StateHandoffChallenge), evaluation.Outcome.GateObserved)
	out, err := submitSatisfied(t, root, "m-skip")
	require.NoError(t, err)
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)
	assert.Contains(t, outcomeFiles(t, root, "m-skip"), "consumed.json", "entering execution consumes the outcome")
}

func reapproveGate(t *testing.T, root, id string) {
	t.Helper()
	for _, event := range []domain.MissionEngineEvent{domain.MissionEventRefinementDone, domain.MissionEventGateApproved} {
		_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(event))
		require.NoError(t, err, event)
	}
}

func TestRequiredChallengePassAuthorizesEntryAndFailureForcesANewGate(t *testing.T) {
	root := handoffRoot(t, "m-req", requiredFacts, analysisOnlyTasks)

	_, err := evaluateHandoff(t, root, "m-req", livemission.ArchivistHandoffInput{})
	require.ErrorContains(t, err, "handoff_challenge_required")
	assert.Empty(t, outcomeFiles(t, root, "m-req"), "a refused evaluation records nothing")

	failed, err := evaluateHandoff(t, root, "m-req", failingChallenge())
	require.NoError(t, err, "a failed challenge is a recorded outcome, not an evaluation error")
	assert.Equal(t, handoff.OutcomeFailed, failed.Outcome.Result)
	assert.True(t, failed.Outcome.Required)
	status := missionStatus(t, root, "m-req")
	assert.Equal(t, domain.StateRefinement, status.State, "a failed handoff returns to refinement")
	assert.Equal(t, 1, status.HandoffAttempt)
	assert.Equal(t, handoff.StatusFailed, status.HandoffStatus)
	assert.Equal(t, handoff.FailureActionReturnToArchivist, status.HandoffNextAction)
	_, err = submitSatisfied(t, root, "m-req")
	require.Error(t, err, "execution needs a new Approval Gate acceptance")

	reapproveGate(t, root, "m-req")
	passed, err := evaluateHandoff(t, root, "m-req", passingChallenge())
	require.NoError(t, err)
	assert.Equal(t, handoff.OutcomePassed, passed.Outcome.Result)
	assert.Equal(t, 2, passed.Outcome.Attempt)
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-req").State, "a passed evaluation does not enter execution by itself")
	out, err := submitSatisfied(t, root, "m-req")
	require.NoError(t, err)
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)
}

// The re-approval gap: before the failure loop was persisted, a failed outcome
// left the mission at the handoff boundary, so an amended package could be
// re-evaluated and enter execution without a new Approval Gate acceptance.
func TestAmendedPackageAfterAFailureCannotEnterExecutionWithoutANewGate(t *testing.T) {
	root := handoffRoot(t, "m-regate", requiredFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-regate", failingChallenge())
	require.NoError(t, err)
	writeTypedPackage(t, root, "m-regate", requiredFacts, analysisOnlyTasks+"- [ ] 1.2 [analysis_artifact] amended after the gate\n")

	_, err = evaluateHandoff(t, root, "m-regate", passingChallenge())
	require.ErrorContains(t, err, "handoff_gate_not_observed")
	_, err = submitSatisfied(t, root, "m-regate")
	require.Error(t, err)
	assert.NotEqual(t, domain.StateExecution, missionStatus(t, root, "m-regate").State)
}

// The Approval Gate must bind the package revision before any handoff outcome
// exists. An amendment made after acceptance but before the first evaluation
// therefore requires a new gate and cannot be evaluated in place.
func TestAmendedPackageBeforeFirstEvaluationCannotUseTheOldGate(t *testing.T) {
	root := handoffRoot(t, "m-gate-digest", informationalFacts, analysisOnlyTasks)
	writeTypedPackage(t, root, "m-gate-digest", informationalFacts, analysisOnlyTasks+"- [ ] 1.2 [analysis_artifact] amended after gate acceptance\n")

	_, err := evaluateHandoff(t, root, "m-gate-digest", livemission.ArchivistHandoffInput{})

	require.ErrorContains(t, err, "handoff_package_changed_after_gate")
	assert.Empty(t, outcomeFiles(t, root, "m-gate-digest"), "a stale gate must not create an authorizing outcome")
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-gate-digest").State)
}

func TestLastFailedAttemptBlocksTheMission(t *testing.T) {
	root := handoffRoot(t, "m-exhaust", requiredFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-exhaust", failingChallenge())
	require.NoError(t, err)
	reapproveGate(t, root, "m-exhaust")

	_, err = evaluateHandoff(t, root, "m-exhaust", failingChallenge())
	require.NoError(t, err)

	status := missionStatus(t, root, "m-exhaust")
	assert.Equal(t, domain.StateBlocked, status.State)
	assert.Equal(t, 2, status.HandoffAttempt)
}

func TestPackageChangedAfterAPassedOutcomeMustReturnToRefinement(t *testing.T) {
	root := handoffRoot(t, "m-after-pass", requiredFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-after-pass", passingChallenge())
	require.NoError(t, err)
	writeTypedPackage(t, root, "m-after-pass", requiredFacts, analysisOnlyTasks+"- [ ] 1.2 [analysis_artifact] changed after the outcome\n")

	_, err = evaluateHandoff(t, root, "m-after-pass", passingChallenge())

	require.ErrorContains(t, err, "handoff_package_changed_after_gate")
	_, err = submitSatisfied(t, root, "m-after-pass")
	require.ErrorContains(t, err, "handoff_package_changed_after_gate")
}

func TestAuthoredPackageRepairInvalidatesGateAndHandoffEvidence(t *testing.T) {
	root := handoffRoot(t, "m-repair", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-repair", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	_, err = submitSatisfied(t, root, "m-repair")
	require.NoError(t, err)

	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	originalDigest, err := handoff.PackageDigest(filepath.Join(basePath, "refined", "m-repair"))
	require.NoError(t, err)
	writeTypedPackage(t, root, "m-repair", informationalFacts, "- [ ] 1.1 [documentation_target] Write the guide without an explicit path\n")
	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-repair", "--event", string(domain.MissionEventRefinementArtifactInvalid))
	require.NoError(t, err)

	status := missionStatus(t, root, "m-repair")
	assert.Equal(t, domain.StateRefinement, status.State)
	assert.Empty(t, status.ApprovalGatePackageDigest)
	assert.Empty(t, status.HandoffStatus)
	assert.Equal(t, "reapprove_gate", status.HandoffNextAction)
	tombstone := filepath.Join(basePath, "refined", "m-repair", ".repair-evidence", "repair-001.json")
	assert.FileExists(t, tombstone)
	tombstoneRaw, err := os.ReadFile(tombstone)
	require.NoError(t, err)
	assert.Contains(t, string(tombstoneRaw), originalDigest)
	assert.Contains(t, string(tombstoneRaw), `"original_state": "EXECUTION"`)
	assert.Contains(t, string(tombstoneRaw), `"result_state": "REFINEMENT"`)
	assert.FileExists(t, filepath.Join(root, "missions", "handoff", "m-repair", "invalidated-001.json"))

	writeTypedPackage(t, root, "m-repair", informationalFacts, "- [ ] 1.1 [documentation_target] Write `docs/repaired.md` after the repair\n")
	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-repair", "--event", string(domain.MissionEventRefinementDone))
	require.NoError(t, err)
	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-repair", "--event", string(domain.MissionEventGateApproved))
	require.NoError(t, err)
	_, err = evaluateHandoff(t, root, "m-repair", livemission.ArchivistHandoffInput{})
	require.NoError(t, err, "the new gate permits a new handoff outcome")
	_, err = submitSatisfied(t, root, "m-repair")
	require.NoError(t, err, "the corrected package enters execution only after new gate and handoff evidence")
}

func TestAuthoredPackageRepairRejectsAValidPackage(t *testing.T) {
	root := handoffRoot(t, "m-repair-valid", informationalFacts, analysisOnlyTasks)
	_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-repair-valid", "--event", string(domain.MissionEventRefinementArtifactInvalid))
	require.ErrorContains(t, err, "requires a malformed documentation_target")
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-repair-valid").State)
}

func TestExecutionPreflightRejectsMalformedTargetBeforeStateOrOutcomeMutation(t *testing.T) {
	root := handoffRoot(t, "m-preflight-target", informationalFacts, "- [ ] 1.1 [documentation_target] Write `docs/valid.md`\n")
	_, err := evaluateHandoff(t, root, "m-preflight-target", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)

	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	badTasks := filepath.Join(basePath, "refined", "m-preflight-target", "tasks.md")
	require.NoError(t, os.WriteFile(badTasks, []byte("- [ ] 1.1 [documentation_target] Write it\n"), 0o600))

	_, err = submitSatisfied(t, root, "m-preflight-target")
	require.ErrorContains(t, err, "explicit backtick-quoted repository-relative path")
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-preflight-target").State)
	assert.NotContains(t, outcomeFiles(t, root, "m-preflight-target"), "consumed.json")
}

func TestExecutionEntryDeniesAnOutcomeTheMissionStateDoesNotRecord(t *testing.T) {
	root := handoffRoot(t, "m-drift", informationalFacts, analysisOnlyTasks)
	store := handoff.NewOutcomeStore(root)
	evaluation, err := evaluateHandoff(t, root, "m-drift", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	// Simulate a mission state that lost its record of the attempt.
	deps := lifecycleDeps(t)
	status := missionStatus(t, root, "m-drift")
	status.HandoffAttempt, status.HandoffStatus, status.HandoffNextAction = 0, "", ""
	require.NoError(t, deps.Save(root, status))
	_, err = store.Latest("m-drift")
	require.NoError(t, err)

	_, err = submitSatisfied(t, root, "m-drift")

	require.ErrorContains(t, err, "handoff_outcome_state_mismatch")
	assert.Equal(t, handoff.OutcomeSkipped, evaluation.Outcome.Result)
}

func TestInterruptedStateSaveIsReconciledBeforeAnyNewEvaluation(t *testing.T) {
	root := handoffRoot(t, "m-reconcile", requiredFacts, analysisOnlyTasks)
	failed, err := evaluateHandoff(t, root, "m-reconcile", failingChallenge())
	require.NoError(t, err)
	// The outcome is durable but the mission state save was lost: the mission is
	// back at the handoff boundary with no recorded attempt.
	deps := lifecycleDeps(t)
	require.NoError(t, deps.Save(root, domain.MissionEngineStatus{MissionID: "m-reconcile", Phase: domain.PhaseApprovalGate, State: domain.StateHandoffChallenge}))
	writeTypedPackage(t, root, "m-reconcile", requiredFacts, analysisOnlyTasks+"- [ ] 1.2 [analysis_artifact] repaired without a new gate\n")

	_, err = evaluateHandoff(t, root, "m-reconcile", passingChallenge())

	require.ErrorContains(t, err, "handoff_gate_not_observed", "the pending failure is applied first, so the repair cannot be evaluated in place")
	status := missionStatus(t, root, "m-reconcile")
	assert.Equal(t, domain.StateRefinement, status.State)
	assert.Equal(t, failed.Outcome.Attempt, status.HandoffAttempt)
}

func TestLowRiskLabelNeverSkipsAPackageWithAConcreteRequireFact(t *testing.T) {
	root := handoffRoot(t, "m-low", requiredFacts, implementationTasks)

	_, err := evaluateHandoff(t, root, "m-low", livemission.ArchivistHandoffInput{RiskLevel: "low"})

	require.ErrorContains(t, err, "handoff_challenge_required")
}

func TestDirectEventWithoutACorrelatedOutcomeIsRejected(t *testing.T) {
	root := handoffRoot(t, "m-direct", informationalFacts, analysisOnlyTasks)

	_, err := submitSatisfied(t, root, "m-direct")

	require.ErrorContains(t, err, "handoff_outcome_missing")
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-direct").State)
}

func TestObsoletePassedEventIsRejectedByTheCommand(t *testing.T) {
	root := handoffRoot(t, "m-old", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-old", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-old", "--event", "handoff_challenge_passed")

	require.ErrorContains(t, err, "unsupported_event")
	require.ErrorContains(t, err, "handoff_challenge_satisfied")
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-old").State)
}

func TestAmendingThePackageAfterTheOutcomeMakesItStale(t *testing.T) {
	root := handoffRoot(t, "m-stale", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-stale", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	writeTypedPackage(t, root, "m-stale", informationalFacts, analysisOnlyTasks+"- [ ] 1.2 [analysis_artifact] one more item\n")

	_, err = submitSatisfied(t, root, "m-stale")

	require.ErrorContains(t, err, "handoff_package_changed_after_gate")
}

func TestAmendingFactsSoTheSkipIsNoLongerAuthorizedDeniesEntry(t *testing.T) {
	root := handoffRoot(t, "m-facts", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-facts", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	writeTypedPackage(t, root, "m-facts", requiredFacts, analysisOnlyTasks)

	_, err = submitSatisfied(t, root, "m-facts")

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "handoff_package_changed_after_gate") || strings.Contains(err.Error(), "handoff_outcome_skip_not_authorized"), err.Error())
}

func TestTamperedOutcomeDeniesEntry(t *testing.T) {
	root := handoffRoot(t, "m-tamper", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-tamper", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	path := filepath.Join(root, "missions", "handoff", "m-tamper", "attempt-001.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var record map[string]any
	require.NoError(t, json.Unmarshal(raw, &record))
	record["required"] = true
	edited, err := json.Marshal(record)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, edited, 0o600))

	_, err = submitSatisfied(t, root, "m-tamper")

	require.ErrorContains(t, err, "handoff_outcome_tampered")
}

func TestOutcomeCopiedFromAnotherMissionDeniesEntry(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-a")
	advanceToHandoffChallenge(t, root, "m-b")
	writeTypedPackage(t, root, "m-a", informationalFacts, analysisOnlyTasks)
	writeTypedPackage(t, root, "m-b", informationalFacts, analysisOnlyTasks)
	_, err := evaluateHandoff(t, root, "m-a", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(root, "missions", "handoff", "m-a", "attempt-001.json"))
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions", "handoff", "m-b"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", "handoff", "m-b", "attempt-001.json"), raw, 0o600))

	_, err = submitSatisfied(t, root, "m-b")

	require.ErrorContains(t, err, "handoff_outcome_cross_mission")
}

func TestReplayingAConsumedOutcomeIsDenied(t *testing.T) {
	root := handoffRoot(t, "m-replay", informationalFacts, analysisOnlyTasks)
	evaluation, err := evaluateHandoff(t, root, "m-replay", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	_, err = submitSatisfied(t, root, "m-replay")
	require.NoError(t, err)

	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	status := missionStatus(t, root, "m-replay")
	_, err = livemission.AuthorizeHandoffExecution(root, basePath, status)

	require.ErrorContains(t, err, "handoff_outcome_replayed")
	_, err = evaluateHandoff(t, root, "m-replay", livemission.ArchivistHandoffInput{})
	require.Error(t, err, "an executing mission cannot be re-evaluated")
	assert.Equal(t, handoff.OutcomeSkipped, evaluation.Outcome.Result)
}

func TestPersistenceFailureEmitsNoAuthorizingOutcomeAndADeterministicError(t *testing.T) {
	root := handoffRoot(t, "m-persist", informationalFacts, analysisOnlyTasks)
	if os.Geteuid() == 0 {
		t.Skip("directory permissions do not constrain root")
	}
	// A read-only outcome directory makes the append fail after evaluation succeeds.
	handoffDir := filepath.Join(root, "missions", "handoff")
	require.NoError(t, os.MkdirAll(handoffDir, 0o750))
	require.NoError(t, os.Chmod(handoffDir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(handoffDir, 0o750) }) //nolint:errcheck // best-effort restore so the temp dir can be removed

	_, err := evaluateHandoff(t, root, "m-persist", livemission.ArchivistHandoffInput{})

	require.ErrorContains(t, err, "handoff_outcome_persist_failed")
	_, err = submitSatisfied(t, root, "m-persist")
	require.Error(t, err)
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-persist").State)
}

func TestApprovalGateIsIndependentOfTheHandoffOutcome(t *testing.T) {
	t.Run("satisfied outcome without an accepted gate", func(t *testing.T) {
		root := setupViewRoot(t, domain.MissionEngineStatus{})
		_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-nogate")
		require.NoError(t, err)
		writeDiscoveryArtifact(t, root, "m-nogate")
		writeRefinedPackage(t, root, "m-nogate")
		for _, event := range []domain.MissionEngineEvent{domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone, domain.MissionEventRefinementDone} {
			_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-nogate", "--event", string(event))
			require.NoError(t, err)
		}
		writeTypedPackage(t, root, "m-nogate", informationalFacts, analysisOnlyTasks)

		_, err = evaluateHandoff(t, root, "m-nogate", livemission.ArchivistHandoffInput{})
		require.ErrorContains(t, err, "handoff_gate_not_observed")
		_, err = submitSatisfied(t, root, "m-nogate")
		require.Error(t, err)
	})
	t.Run("accepted gate without an outcome", func(t *testing.T) {
		root := handoffRoot(t, "m-noout", informationalFacts, analysisOnlyTasks)
		_, err := submitSatisfied(t, root, "m-noout")
		require.ErrorContains(t, err, "handoff_outcome_missing")
	})
	t.Run("analysis-only terminal route bypasses the handoff boundary", func(t *testing.T) {
		root := setupViewRoot(t, domain.MissionEngineStatus{})
		_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-analysis")
		require.NoError(t, err)
		writeDiscoveryArtifact(t, root, "m-analysis")
		writeRefinedPackage(t, root, "m-analysis")
		for _, event := range []domain.MissionEngineEvent{domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone, domain.MissionEventRefinementDone} {
			_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-analysis", "--event", string(event))
			require.NoError(t, err)
		}
		writeRefinedPackage(t, root, "m-analysis")

		_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-analysis", "--event", string(domain.MissionEventGateApprovedAnalysisOnly))

		require.NoError(t, err)
		assert.Empty(t, outcomeFiles(t, root, "m-analysis"))
	})
}

func metadataTasks(required string) string {
	return analysisOnlyTasks + "\n```yaml\nhandoff_verification:\n  enabled: " + required + "\n  required: " + required + "\n```\n"
}

func TestOptionalHandoffMetadataMustMirrorTheDerivedPolicy(t *testing.T) {
	t.Run("a mirror that agrees is accepted but proves nothing", func(t *testing.T) {
		root := handoffRoot(t, "m-meta-ok", informationalFacts, metadataTasks("false"))

		_, err := evaluateHandoff(t, root, "m-meta-ok", livemission.ArchivistHandoffInput{})
		require.NoError(t, err)

		_, err = submitSatisfied(t, root, "m-meta-ok")
		require.NoError(t, err)
	})
	t.Run("metadata claiming the challenge is not required cannot skip a required one", func(t *testing.T) {
		root := handoffRoot(t, "m-meta-low", requiredFacts, metadataTasks("false"))

		_, err := evaluateHandoff(t, root, "m-meta-low", passingChallenge())

		require.ErrorContains(t, err, "handoff_metadata_mismatch")
		assert.Empty(t, outcomeFiles(t, root, "m-meta-low"))
	})
	t.Run("metadata edited after the outcome fails entry closed", func(t *testing.T) {
		root := handoffRoot(t, "m-meta-edit", informationalFacts, analysisOnlyTasks)
		_, err := evaluateHandoff(t, root, "m-meta-edit", livemission.ArchivistHandoffInput{})
		require.NoError(t, err)
		_, basePath, err := cliutil.ResolveActiveBasePath(root)
		require.NoError(t, err)
		tasks := filepath.Join(basePath, "refined", "m-meta-edit", "tasks.md")
		require.NoError(t, os.WriteFile(tasks, []byte(metadataTasks("true")), 0o600))

		_, err = submitSatisfied(t, root, "m-meta-edit")

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "handoff_metadata_mismatch") || strings.Contains(err.Error(), "handoff_package_changed_after_gate"), err.Error())
	})
	t.Run("non-boolean metadata is rejected", func(t *testing.T) {
		root := handoffRoot(t, "m-meta-bad", informationalFacts, analysisOnlyTasks+"\n```yaml\nhandoff_verification:\n  required: maybe\n```\n")

		_, err := evaluateHandoff(t, root, "m-meta-bad", livemission.ArchivistHandoffInput{})

		require.ErrorContains(t, err, "handoff_metadata_invalid")
	})
}

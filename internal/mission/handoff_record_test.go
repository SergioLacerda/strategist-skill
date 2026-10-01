package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recordFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: ["which store?"]
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: false
`

const recordInformationalFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
`

type recordFixture struct {
	root, basePath, id string
	engine             *domain.MissionEngine
}

func newRecordFixture(t *testing.T, id, facts string) recordFixture {
	t.Helper()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: id})
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone, domain.MissionEventGateApproved,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err)
	}
	fixture := recordFixture{root: t.TempDir(), basePath: t.TempDir(), id: id, engine: engine}
	fixture.writePackage(t, facts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	return fixture
}

func (f recordFixture) writePackage(t *testing.T, facts, tasks string) {
	t.Helper()
	dir := filepath.Join(f.basePath, "refined", f.id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	files := map[string]string{
		"analysis.md": "---\nmission_id: " + f.id + "\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tasks,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}

func recordChallenges() ArchivistHandoffInput {
	no := false
	return ArchivistHandoffInput{
		Challenges: []handoff.Challenge{
			{ID: "HC-001", Type: handoff.ChallengeObjective, SourceRefs: []string{"G-001"}, Critical: true},
			{ID: "HC-002", Type: handoff.ChallengeBoundary, SourceRefs: []string{"X-001"}, Critical: true},
			{ID: "HC-003", Type: handoff.ChallengeClassification, SourceRefs: []string{"D-001", "Q-001"}, Critical: true,
				ExpectedClassification: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved}},
			{ID: "HC-004", Type: handoff.ChallengeGate, SourceRefs: []string{"approval.required"}, Critical: true, ExpectedGateAllowed: &no},
		},
		Ack: handoff.Acknowledgment{
			ChallengeRefs:   []string{"HC-001", "HC-002", "HC-003", "HC-004"},
			UnderstoodRefs:  []string{"G-001", "X-001", "D-001", "Q-001", "approval.required"},
			Classifications: map[string]string{"D-001": handoff.DecisionApproved, "Q-001": handoff.QuestionUnresolved},
			GateAllowed:     &no,
		},
	}
}

func openQuestionSummary(agent string) *domain.ConfidenceSummary {
	return &domain.ConfidenceSummary{
		PolicyVersion: domain.ConfidencePolicyVersion, CalibrationStatus: domain.CalibrationNoSample,
		OpenQuestions: []domain.ConfidenceClaim{{
			ID: "Q-001", Statement: "Is the handoff complete?", Agent: agent, CorrelationKey: "question-1",
			ClaimKind: domain.ClaimKindQuestion, ConfidencePercent: 40,
		}},
	}
}

func TestRecordArchivistHandoffRecordsAMissingConfidenceWhenNoneIsSupplied(t *testing.T) {
	f := newRecordFixture(t, "rec-missing", recordFacts)

	evaluation, status, changed, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, recordChallenges())

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, handoff.OutcomePassed, evaluation.Outcome.Result)
	assert.Equal(t, domain.StateHandoffChallenge, status.State)
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(f.root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, telemetry.ConfidenceCoverageMissing, records[0].CoverageStatus)
	assert.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, records[0].Agent)
	assert.Equal(t, "handoff-attempt-1", records[0].Run, "the record is scoped to its attempt")
	history, err := telemetry.ReadHandoffChallenges(telemetry.HandoffChallengeHistoryPath(f.root))
	require.NoError(t, err)
	require.Len(t, history, 1)
}

func TestRecordArchivistHandoffPersistsDeclaredOpenQuestions(t *testing.T) {
	f := newRecordFixture(t, "rec-question", recordFacts)
	input := recordChallenges()
	input.ConfidenceSummary = openQuestionSummary("archivist")

	_, _, _, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)

	require.NoError(t, err)
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(f.root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "Q-001", records[0].ClaimID)
	assert.Equal(t, domain.ClaimKindQuestion, records[0].ClaimKind)
}

// Characterizes AR-008: a confidence summary carrying another role's claim is
// not rejected; it is recorded under the fixed handoff_challenge producer, not
// the claim's own declared agent. Pinned so changing it is a deliberate decision.
func TestRecordArchivistHandoffRecordsAForeignAgentClaimUnderTheHandoffChallengeProducer(t *testing.T) {
	f := newRecordFixture(t, "rec-foreign", recordFacts)
	input := recordChallenges()
	input.ConfidenceSummary = openQuestionSummary("ranger")

	_, _, _, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)

	require.NoError(t, err)
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(f.root))
	require.NoError(t, err)
	require.Len(t, records, 1, "the foreign-agent claim is recorded, not rejected")
	assert.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, records[0].Agent)
}

func TestRecordArchivistHandoffRejectsAnInvalidConfidenceSummaryAndLeavesNoOutcome(t *testing.T) {
	f := newRecordFixture(t, "rec-invalid", recordFacts)
	input := recordChallenges()
	input.ConfidenceSummary = &domain.ConfidenceSummary{PolicyVersion: domain.ConfidencePolicyVersion, CalibrationStatus: domain.CalibrationNoSample}

	_, status, changed, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)

	require.ErrorContains(t, err, "handoff_outcome_persist_failed")
	assert.False(t, changed)
	assert.Equal(t, 0, status.HandoffAttempt)
	_, latestErr := handoff.NewOutcomeStore(f.root).Latest(f.id)
	require.ErrorContains(t, latestErr, "handoff_outcome_missing")
}

func TestRecordArchivistHandoffFailureReturnsToRefinementAndRecordsTheLoop(t *testing.T) {
	f := newRecordFixture(t, "rec-fail", recordFacts)
	input := recordChallenges()
	yes := true
	input.Ack.GateAllowed = &yes

	evaluation, status, changed, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, input)

	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, handoff.OutcomeFailed, evaluation.Outcome.Result)
	assert.Equal(t, domain.StateRefinement, status.State)
	assert.Equal(t, 1, status.HandoffAttempt)
	assert.Equal(t, handoff.FailureActionReturnToArchivist, status.HandoffNextAction)
	_, _, _, err = RecordArchivistHandoff(f.root, f.basePath, f.engine, recordChallenges())
	require.ErrorContains(t, err, "handoff_gate_not_observed", "no evaluation is possible until the gate is accepted again")
}

func TestRecordArchivistHandoffSkipRecordsTheAttemptWithoutEnteringExecution(t *testing.T) {
	f := newRecordFixture(t, "rec-skip", recordInformationalFacts)

	evaluation, status, _, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, ArchivistHandoffInput{})

	require.NoError(t, err)
	assert.Equal(t, handoff.OutcomeSkipped, evaluation.Outcome.Result)
	assert.Equal(t, domain.StateHandoffChallenge, status.State)
	assert.Equal(t, 1, status.HandoffAttempt)
}

func TestRecordArchivistHandoffPersistenceFailureDoesNotAdvanceTheMission(t *testing.T) {
	f := newRecordFixture(t, "rec-persist", recordInformationalFacts)
	require.NoError(t, os.MkdirAll(filepath.Join(f.root, "missions"), 0o755))
	// A regular file where the outcome directory must be makes the append fail.
	require.NoError(t, os.MkdirAll(filepath.Join(f.root, "missions", "handoff"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(f.root, "missions", "handoff", f.id), []byte("blocked"), 0o644))
	before := f.engine.Status()

	_, status, changed, err := RecordArchivistHandoff(f.root, f.basePath, f.engine, ArchivistHandoffInput{})

	require.Error(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, status)
}

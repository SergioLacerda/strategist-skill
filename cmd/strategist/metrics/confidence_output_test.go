package metrics

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

const oneQuestionBatch = batchClaimsYAML

const noQuestionBatch = `claims:
  - {id: C-1, statement: s1, agent: ranger, correlation_key: k1, claim_kind: assertion, confidence_percent: 90, evidence_ids: [E-1], evidence_classes: [explicit]}
evidence:
  - {id: E-1, source_ref: a.go, class: explicit, confidence: high}
`

func confidenceOutput(t *testing.T, root, mission string) string {
	t.Helper()
	cmd, out := testCommand("metrics")
	require.NoError(t, RunConfidence(cmd, testDependencies(), root, mission))
	return out.String()
}

// fixedHistory is the scratch history the M016 pins below read: mission m holds
// three ranger claims (two assertions, one question); mission z holds one
// assertion and no question.
func fixedHistory(t *testing.T) string {
	t.Helper()
	root := testRoot(t)
	_, err := recordBatch(t, root, "ranger", oneQuestionBatch)
	require.NoError(t, err)
	cmd, _ := testCommand("metrics")
	cmd.SetIn(strings.NewReader(noQuestionBatch))
	require.NoError(t, RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "z", Agent: "ranger", ClaimFile: "-"}))
	return root
}

// M016 pins: the exact output of `metrics confidence` for the fixed history. Every
// later change of this command must leave these bytes unchanged, except the one
// intended difference DEC-004 names: a mission (and an agent) with no questions
// prints question_preservation_rate as n/a instead of 0.00. The JSON field and the
// computed metric are unchanged.
func TestConfidenceOutputForAMissionWithQuestionsIsPinned(t *testing.T) {
	require.Equal(t, wantMissionM, confidenceOutput(t, fixedHistory(t), "m"))
}

func TestConfidenceOutputWithoutAMissionIsPinned(t *testing.T) {
	require.Equal(t, wantAllMissions, confidenceOutput(t, fixedHistory(t), ""))
}

func TestConfidenceOutputForAMissionWithoutQuestionsIsPinned(t *testing.T) {
	require.Equal(t, wantMissionZ, confidenceOutput(t, fixedHistory(t), "z"))
}

const wantMissionM = `policy_version: v1
confidence_distribution.low: 1
confidence_distribution.medium: 1
confidence_distribution.high: 1
claim_kinds.question: 1
claim_kinds.assertion: 2
assertion_evidence_coverage: 1.00
unsupported_assertion_rate: 0.00
question_preservation_rate: 1.00
corrected_high_confidence_claim_rate: 0.00
sample_size: 3
ground_truth_sample_size: 0
missing_records: 0
rejected_records: 0
duplicate_records: 0
review_required: true
confidence_unavailable: false
calibration_status: uncalibrated
agent.ranger.sample_size: 3
agent.ranger.assertion_evidence_coverage: 1.00
agent.ranger.unsupported_assertion_rate: 0.00
agent.ranger.question_preservation_rate: 1.00
agent.ranger.corrected_high_confidence_claim_rate: 0.00
agent.ranger.ground_truth_sample_size: 0
agent.ranger.calibration_status: uncalibrated
gate_outcome: none
`

const wantMissionZ = `policy_version: v1
confidence_distribution.low: 0
confidence_distribution.medium: 0
confidence_distribution.high: 1
claim_kinds.question: 0
claim_kinds.assertion: 1
assertion_evidence_coverage: 1.00
unsupported_assertion_rate: 0.00
question_preservation_rate: n/a
corrected_high_confidence_claim_rate: 0.00
sample_size: 1
ground_truth_sample_size: 0
missing_records: 0
rejected_records: 0
duplicate_records: 0
review_required: true
confidence_unavailable: false
calibration_status: uncalibrated
agent.ranger.sample_size: 1
agent.ranger.assertion_evidence_coverage: 1.00
agent.ranger.unsupported_assertion_rate: 0.00
agent.ranger.question_preservation_rate: n/a
agent.ranger.corrected_high_confidence_claim_rate: 0.00
agent.ranger.ground_truth_sample_size: 0
agent.ranger.calibration_status: uncalibrated
gate_outcome: none
`

const wantAllMissions = `policy_version: v1
confidence_distribution.low: 1
confidence_distribution.medium: 1
confidence_distribution.high: 2
claim_kinds.question: 1
claim_kinds.assertion: 3
assertion_evidence_coverage: 1.00
unsupported_assertion_rate: 0.00
question_preservation_rate: 1.00
corrected_high_confidence_claim_rate: 0.00
sample_size: 4
ground_truth_sample_size: 0
missing_records: 0
rejected_records: 0
duplicate_records: 0
review_required: true
confidence_unavailable: false
calibration_status: uncalibrated
agent.ranger.sample_size: 4
agent.ranger.assertion_evidence_coverage: 1.00
agent.ranger.unsupported_assertion_rate: 0.00
agent.ranger.question_preservation_rate: 1.00
agent.ranger.corrected_high_confidence_claim_rate: 0.00
agent.ranger.ground_truth_sample_size: 0
agent.ranger.calibration_status: uncalibrated
`

func TestZeroQuestionRateIsDisplayedAsNotApplicableButComputedAsBefore(t *testing.T) {
	review, err := telemetry.LoadConfidenceGateReview(fixedHistory(t), "z")
	require.NoError(t, err)

	require.Zero(t, review.Metrics.QuestionPreservationRate, "the metric itself is unchanged")
	require.Zero(t, review.Metrics.ClaimKinds["question"])
	var out strings.Builder
	require.NoError(t, PrintConfidenceMetrics(&out, review))
	require.Contains(t, out.String(), "\nquestion_preservation_rate: n/a\n")
	require.Contains(t, out.String(), "\nagent.ranger.question_preservation_rate: n/a\n")
}

func TestQuestionRateIsDisplayedAsANumberWhenQuestionsExist(t *testing.T) {
	review, err := telemetry.LoadConfidenceGateReview(fixedHistory(t), "m")
	require.NoError(t, err)

	var out strings.Builder
	require.NoError(t, PrintConfidenceMetrics(&out, review))
	require.Contains(t, out.String(), "\nquestion_preservation_rate: 1.00\n")
	require.Contains(t, out.String(), "\nagent.ranger.question_preservation_rate: 1.00\n")
}

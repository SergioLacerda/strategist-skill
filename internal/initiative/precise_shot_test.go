package initiative

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func preciseResult(advice Advice, status CheckStatus, evidence string) Result {
	refs := []EvidenceRef{{ID: evidence, Class: "explicit"}}
	return Result{
		ResultID: resultID(advice, 1), Sequence: 1, AdviceID: advice.AdviceID, MissionID: advice.MissionID,
		Role: advice.Role, RunID: advice.RunID, GateIndependent: true,
		Checks:       []ObligationCheck{{ID: advice.Diligence.Checks[0], Status: status, EvidenceRefs: refs}},
		EvidenceRefs: refs,
	}
}

func TestPreciseShotUsesStableIdentityAndLocalizedLabel(t *testing.T) {
	require.Equal(t, "PRECISE-SHOT", PreciseShotMechanism)
	require.Equal(t, "TIRO PRECISO", PreciseShotLabel("pt-BR"))
	require.Equal(t, "PRECISE-SHOT", PreciseShotLabel("en-US"))
}

func TestPreciseShotCompleteResultReplaysDeterministically(t *testing.T) {
	advice := validAdvice()
	result := preciseResult(advice, CheckSatisfied, "e-precise")
	first, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	second, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.Equal(t, ConfidenceHigh, first.Assessed)
	require.Equal(t, ConfidenceHigh, first.Effective)
	require.Equal(t, PreciseShotCalibrationNoSample, first.CalibrationStatus)
	require.Equal(t, first.AssessmentID, second.AssessmentID)
	require.Equal(t, first.InputDigest, second.InputDigest)
	require.Equal(t, first.EvidenceSummary, second.EvidenceSummary)
	require.NoError(t, ValidateAssessmentAgainst(advice, result, first))
}

func TestPreciseShotAppliesCeilingWithoutChangingAssessment(t *testing.T) {
	advice := validAdvice()
	advice.Diligence.ConfidenceCeiling = ConfidenceMedium
	result := preciseResult(advice, CheckSatisfied, "e-ceiling")
	assessment, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.Equal(t, ConfidenceHigh, assessment.Assessed)
	require.Equal(t, ConfidenceMedium, assessment.Ceiling)
	require.Equal(t, ConfidenceMedium, assessment.Effective)
	require.False(t, assessment.Challenge)
	require.Contains(t, assessment.Reasons, "confidence_ceiling_applied")
}

func TestPreciseShotDistinguishesPartialAndMissingEvidence(t *testing.T) {
	advice := validAdvice()
	partial := preciseResult(advice, CheckPartial, "e-partial")
	assessment, err := AssessConfidence(advice, partial)
	require.NoError(t, err)
	require.Equal(t, ConfidenceMedium, assessment.Assessed)
	require.Contains(t, assessment.Reasons, "partial_obligation:inspect")

	missing := preciseResult(advice, CheckBlocked, "e-blocked")
	missing.EvidenceRefs = nil
	missing.Checks[0].EvidenceRefs = nil
	assessment, err = AssessConfidence(advice, missing)
	require.NoError(t, err)
	require.Equal(t, ConfidenceLow, assessment.Effective)
	require.Contains(t, assessment.Reasons, "missing_result_evidence")
	require.NotNil(t, assessment.Escalation)
}

func TestBoundEscalationSuppressesReplayAndStopsAtMaximum(t *testing.T) {
	advice := validAdvice()
	assessment, err := AssessConfidence(advice, preciseResult(advice, CheckBlocked, "e-loop"))
	require.NoError(t, err)
	require.NotNil(t, assessment.Escalation)
	policy := DefaultEscalationPolicy()
	first, err := BoundEscalation(*assessment.Escalation, EscalationState{}, policy)
	require.NoError(t, err)
	require.Equal(t, EscalationRequested, first.Status)
	replay, err := BoundEscalation(first, EscalationState{LastIdempotencyKey: first.IdempotencyKey}, policy)
	require.NoError(t, err)
	require.Equal(t, EscalationSuppressed, replay.Status)
	terminal, err := BoundEscalation(first, EscalationState{RequestCount: policy.MaxRequests}, policy)
	require.NoError(t, err)
	require.Equal(t, EscalationHumanReview, terminal.Status)
}

func TestRuntimePersistsAndReadsPreciseShotAssessment(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	advice, _, err := runtime.EnterRole(AdviceInput{MissionID: "persist", Role: "ranger", RunID: "run", Trigger: TriggerInitial})
	require.NoError(t, err)
	result := Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []ObligationCheck{{ID: "inspect_evidence", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}, {ID: "test_alternatives", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-2", Class: "explicit"}}}, {ID: "record_obligations", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-3", Class: "explicit"}}}},
		EvidenceRefs:    []EvidenceRef{{ID: "e-1", Class: "explicit"}, {ID: "e-2", Class: "explicit"}, {ID: "e-3", Class: "explicit"}},
	}
	assessment, err := runtime.RecordResult(advice, result)
	require.NoError(t, err)
	got, found, err := LatestAssessment(filepath.Join(root, "memory", "initiative-records.jsonl"), advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, assessment.AssessmentID, got.AssessmentID)
	legacy, found, err := LatestAssessment(filepath.Join(t.TempDir(), "legacy.jsonl"), advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, legacy.AssessmentID)
}

package handoffconsumer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func confidencePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "memory", "confidence-records.jsonl")
}

func records(t *testing.T, path string) []telemetry.ConfidenceRecord {
	t.Helper()
	got, err := telemetry.ReadConfidenceRecords(path)
	require.NoError(t, err)
	return got
}

func TestAnApprovedReadingBecomesAnAssertionUnderTheHandoffProducer(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.94)), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())
	path := confidencePath(t)

	require.NoError(t, RecordConfidence(path, precheckRequest(), report, "handoff-attempt-1"))
	got := records(t, path)
	require.Len(t, got, 1)
	require.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, got[0].Agent)
	require.Equal(t, domain.ClaimKindAssertion, got[0].ClaimKind)
	require.Equal(t, 94, got[0].ConfidencePercent)
	require.Equal(t, "reported", got[0].CoverageStatus)
	require.Equal(t, "provider-conformance-ranger_to_archivist", got[0].CorrelationKey)
	require.Equal(t, []string{"explicit"}, got[0].EvidenceClasses, "the report itself is explicit evidence; the conformance stays uncalibrated")
	require.Equal(t, domain.CalibrationNoSample, got[0].CalibrationStatus)
}

func TestALowReadingIsAQuestionNeverAManufacturedLowAssertion(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.40, "objective_clear")), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())
	path := confidencePath(t)

	require.NoError(t, RecordConfidence(path, precheckRequest(), report, ""))
	got := records(t, path)
	require.Len(t, got, 1)
	require.Equal(t, domain.ClaimKindQuestion, got[0].ClaimKind, "low-confidence claims must be questions")
	require.Equal(t, "reported", got[0].CoverageStatus)
}

func TestAnEnabledProviderThatDidNotAnswerLeavesAnExplicitMissingRecordWithTheReason(t *testing.T) {
	r := newRig(t, jevtest.Status(503), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())
	path := confidencePath(t)

	require.NoError(t, RecordConfidence(path, precheckRequest(), report, ""))
	got := records(t, path)
	require.Len(t, got, 1)
	require.Equal(t, "missing", got[0].CoverageStatus)
	require.Contains(t, got[0].MissingReason, "unavailable", "an honest reason, never a silent gap")
}

func TestADisabledIntegrationWritesNothing(t *testing.T) {
	path := confidencePath(t)
	require.NoError(t, RecordConfidence(path, precheckRequest(), Report{Status: StatusDisabled}, ""))
	require.Empty(t, records(t, path), "a boundary that is not enabled is not a silent gap")
}

func TestARecordedReuseIsNotAnotherObservation(t *testing.T) {
	path := confidencePath(t)
	report := Report{Status: StatusApproved, Reused: true, Confidence: 0.96}
	require.NoError(t, RecordConfidence(path, precheckRequest(), report, ""))
	require.Empty(t, records(t, path), "the original gate reading already recorded it once")
}

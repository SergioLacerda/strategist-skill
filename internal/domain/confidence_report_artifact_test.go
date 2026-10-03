package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfidenceReportFallbacksAreExplicit(t *testing.T) {
	missing := NewMissingConfidenceReport("provider did not return a report")
	require.NoError(t, missing.Validate())
	assert.Equal(t, ConfidenceReportMissing, missing.Status)

	incompatible := NewIncompatibleConfidenceReport("confidence-report/v2 is unsupported")
	require.NoError(t, incompatible.Validate())
	assert.Equal(t, ConfidenceReportIncompatible, incompatible.Status)
}

func TestConfidenceReportFromSummaryAdaptsAvailableReport(t *testing.T) {
	summary := ConfidenceSummary{
		PolicyVersion: ConfidencePolicyVersion,
		Claims: []ConfidenceClaim{{
			ID: "c-1", Statement: "ready", Agent: "ranger", CorrelationKey: "k-1",
			ClaimKind: ClaimKindAssertion, ConfidencePercent: 80,
			EvidenceIDs: []string{"e-1"}, EvidenceClasses: []string{EvidenceClassExplicit},
			CalibrationStatus: CalibrationNoSample, SampleSize: 0,
		}},
		Evidence:   []Evidence{{ID: "e-1", SourceRef: "finding.md", Class: EvidenceClassExplicit, Confidence: ConfidenceHigh}},
		SampleSize: 0, CalibrationStatus: CalibrationNoSample,
	}
	artifact, err := ConfidenceReportFromSummary(summary, "report-1", "ranger", "run-1", "legacy-adapter")
	require.NoError(t, err)
	require.Equal(t, ConfidenceReportAvailable, artifact.Status)
	require.Equal(t, "report-1", artifact.ReportID)
	require.NoError(t, artifact.Validate())
}

func TestConfidenceReportFromMissingSummaryProducesMissingFallback(t *testing.T) {
	summary := ConfidenceSummary{
		PolicyVersion:     ConfidencePolicyVersion,
		MissingRecord:     true,
		CalibrationStatus: CalibrationNoSample,
	}
	artifact, err := ConfidenceReportFromSummary(summary, "", "sniper", "run-2", "legacy-adapter")
	require.NoError(t, err)
	require.Equal(t, ConfidenceReportMissing, artifact.Status)
	require.Contains(t, artifact.Reason, "missing")
	require.NoError(t, artifact.Validate())
}

func TestConfidenceReportRejectsAvailableWithoutProviderIdentity(t *testing.T) {
	artifact := ConfidenceReportArtifact{
		SchemaVersion:     ConfidenceReportArtifactSchemaVersion,
		Status:            ConfidenceReportAvailable,
		PolicyVersion:     ConfidencePolicyVersion,
		CalibrationStatus: CalibrationNoSample,
	}
	err := artifact.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report_id and source")
}

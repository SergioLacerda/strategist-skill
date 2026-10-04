package telemetry_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestNewConfidenceReportEventProjectsAvailableIdentity(t *testing.T) {
	artifact := domain.ConfidenceReportArtifact{
		SchemaVersion: domain.ConfidenceReportArtifactSchemaVersion,
		Status:        domain.ConfidenceReportAvailable, ReportID: "report-1", Role: "ranger",
		RunID: "run-1", Source: "legacy-adapter", PolicyVersion: domain.ConfidencePolicyVersion,
		CalibrationStatus: domain.CalibrationNoSample,
	}
	event := telemetry.NewConfidenceReportEvent("run-1", artifact)
	require.Equal(t, telemetry.ConfidenceReportEventName, event.Name)
	require.Equal(t, telemetry.SeverityInfo, event.SeverityNumber)
	require.Equal(t, domain.ConfidenceReportArtifactSchemaVersion, event.Attributes[telemetry.AttrSchemaVersion])
	require.Equal(t, domain.CanonicalTaxonomyVersion, event.Attributes[telemetry.AttrTaxonomyVersion])
	require.Equal(t, "available", event.Attributes[telemetry.AttrConfidenceReportStatus])
	require.Equal(t, "report-1", event.Attributes[telemetry.AttrConfidenceReportID])
	require.NotContains(t, event.Attributes, telemetry.AttrConfidenceReportReason)
}

func TestNewConfidenceReportEventElevatesFallbackSeverity(t *testing.T) {
	artifact := domain.NewIncompatibleConfidenceReport("unsupported report contract")
	event := telemetry.NewConfidenceReportEvent("run-2", artifact)
	require.Equal(t, telemetry.SeverityError, event.SeverityNumber)
	require.Equal(t, "incompatible", event.Attributes[telemetry.AttrStatus])
	require.Equal(t, "unsupported report contract", event.Attributes[telemetry.AttrConfidenceReportReason])
}

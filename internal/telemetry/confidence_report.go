package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

const (
	// ConfidenceReportEventName identifies confidence-provider boundary events.
	ConfidenceReportEventName = "strategist.confidence.report"
	// ConfidenceReportContractID identifies the provider-neutral report event.
	ConfidenceReportContractID = "confidence-report/v1"
)

// NewConfidenceReportEvent records only the versioned report identity and
// bounded outcome. It never emits claims, provider payloads, or raw evidence.
func NewConfidenceReportEvent(runID string, artifact domain.ConfidenceReportArtifact) Event {
	severity := SeverityInfo
	if artifact.Status == domain.ConfidenceReportMissing {
		severity = SeverityWarn
	}
	if artifact.Status == domain.ConfidenceReportIncompatible {
		severity = SeverityError
	}
	event := NewEvent(ConfidenceReportEventName, severity, runID, true)
	event.Attributes = map[string]any{
		AttrEventContractID:             ConfidenceReportContractID,
		AttrEventAuthority:              AuthorityStrategistLocal,
		AttrSchemaVersion:               artifact.SchemaVersion,
		AttrTaxonomyVersion:             domain.CanonicalTaxonomyVersion,
		AttrComponent:                   "confidence",
		AttrPhase:                       "confidence",
		AttrStatus:                      string(artifact.Status),
		AttrConfidenceReportStatus:      string(artifact.Status),
		AttrConfidenceReportID:          artifact.ReportID,
		AttrConfidenceReportSource:      artifact.Source,
		AttrConfidencePolicyVersion:     artifact.PolicyVersion,
		AttrConfidenceCalibrationStatus: artifact.CalibrationStatus,
		AttrConfidenceSampleSize:        artifact.SampleSize,
	}
	if artifact.Role != "" {
		event.Attributes[AttrRole] = artifact.Role
	}
	if artifact.RunID != "" {
		event.Attributes[AttrRoleRun] = artifact.RunID
	}
	if artifact.PolicyDigest != "" {
		event.Attributes[AttrInitiativePolicyDigest] = artifact.PolicyDigest
	}
	if artifact.Reason != "" {
		event.Attributes[AttrConfidenceReportReason] = artifact.Reason
	}
	return event
}

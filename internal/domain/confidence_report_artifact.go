package domain

import (
	"fmt"
	"strings"
)

// ConfidenceReportArtifactSchemaVersion identifies the provider-neutral
// confidence boundary. A provider may produce the report, but it cannot become
// a policy or execution authority through this artifact.
const ConfidenceReportArtifactSchemaVersion = "strategist-confidence-report/v1"

// ConfidenceReportStatus describes whether a confidence provider returned a
// compatible report. Missing and incompatible are intentional states, never an
// empty successful report.
type ConfidenceReportStatus string

const (
	// ConfidenceReportAvailable indicates that a compatible report is present.
	ConfidenceReportAvailable ConfidenceReportStatus = "available"
	// ConfidenceReportMissing indicates that no report was produced.
	ConfidenceReportMissing ConfidenceReportStatus = "missing"
	// ConfidenceReportIncompatible indicates that the report cannot be consumed.
	ConfidenceReportIncompatible ConfidenceReportStatus = "incompatible"
)

// ConfidenceReportArtifact is the provider-neutral handoff from a confidence
// producer to Strategist. It carries summary-level calibration only; detailed
// claims remain in ConfidenceSummary and their existing schemas.
type ConfidenceReportArtifact struct {
	SchemaVersion     string                 `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion   string                 `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Status            ConfidenceReportStatus `json:"status" yaml:"status"`
	ReportID          string                 `json:"report_id,omitempty" yaml:"report_id,omitempty"`
	Role              string                 `json:"role,omitempty" yaml:"role,omitempty"`
	RunID             string                 `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	Source            string                 `json:"source,omitempty" yaml:"source,omitempty"`
	PolicyVersion     string                 `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest      string                 `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
	ConfidenceLevel   string                 `json:"confidence_level,omitempty" yaml:"confidence_level,omitempty"`
	CalibrationStatus string                 `json:"calibration_status,omitempty" yaml:"calibration_status,omitempty"`
	SampleSize        int                    `json:"sample_size,omitempty" yaml:"sample_size,omitempty"`
	Reason            string                 `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// NewMissingConfidenceReport creates an explicit absence fallback.
func NewMissingConfidenceReport(reason string) ConfidenceReportArtifact {
	return ConfidenceReportArtifact{
		SchemaVersion:   ConfidenceReportArtifactSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		Status:          ConfidenceReportMissing,
		Reason:          strings.TrimSpace(reason),
	}
}

// NewIncompatibleConfidenceReport creates an explicit compatibility fallback.
func NewIncompatibleConfidenceReport(reason string) ConfidenceReportArtifact {
	return ConfidenceReportArtifact{
		SchemaVersion:   ConfidenceReportArtifactSchemaVersion,
		TaxonomyVersion: CanonicalTaxonomyVersion,
		Status:          ConfidenceReportIncompatible,
		Reason:          strings.TrimSpace(reason),
	}
}

// ConfidenceReportFromSummary adapts the current handoff summary to the
// provider-neutral artifact without changing the summary's persisted shape.
func ConfidenceReportFromSummary(summary ConfidenceSummary, reportID, role, runID, source string) (ConfidenceReportArtifact, error) {
	if err := ValidateConfidenceSummary(summary); err != nil {
		return ConfidenceReportArtifact{}, fmt.Errorf("confidence report: %w", err)
	}
	if summary.MissingRecord {
		artifact := NewMissingConfidenceReport("confidence summary is missing")
		artifact.Role, artifact.RunID, artifact.Source = strings.TrimSpace(role), strings.TrimSpace(runID), strings.TrimSpace(source)
		return artifact, nil
	}
	artifact := ConfidenceReportArtifact{
		SchemaVersion:     ConfidenceReportArtifactSchemaVersion,
		TaxonomyVersion:   CanonicalTaxonomyVersion,
		Status:            ConfidenceReportAvailable,
		ReportID:          strings.TrimSpace(reportID),
		Role:              strings.TrimSpace(role),
		RunID:             strings.TrimSpace(runID),
		Source:            strings.TrimSpace(source),
		PolicyVersion:     strings.TrimSpace(summary.PolicyVersion),
		CalibrationStatus: strings.TrimSpace(summary.CalibrationStatus),
		SampleSize:        summary.SampleSize,
	}
	if err := artifact.Validate(); err != nil {
		return ConfidenceReportArtifact{}, err
	}
	return artifact, nil
}

// Validate checks identity, status, and calibration without granting the
// report authority over Approval Gates, Roles, Weapons, or Stages.
func (a ConfidenceReportArtifact) Validate() error {
	if a.SchemaVersion != ConfidenceReportArtifactSchemaVersion {
		return fmt.Errorf("confidence report: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("confidence report: %w", err)
	}
	if err := validateConfidenceReportStatus(a); err != nil {
		return err
	}
	if err := validateConfidenceReportCalibration(a); err != nil {
		return err
	}
	return nil
}

func validateConfidenceReportStatus(a ConfidenceReportArtifact) error {
	switch a.Status {
	case ConfidenceReportAvailable:
		return validateAvailableConfidenceReport(a)
	case ConfidenceReportMissing, ConfidenceReportIncompatible:
		return validateFallbackConfidenceReport(a)
	default:
		return fmt.Errorf("confidence report: unsupported status %q", a.Status)
	}
}

func validateAvailableConfidenceReport(a ConfidenceReportArtifact) error {
	if strings.TrimSpace(a.ReportID) == "" || strings.TrimSpace(a.Source) == "" {
		return fmt.Errorf("confidence report: available status requires report_id and source")
	}
	if strings.TrimSpace(a.PolicyVersion) == "" {
		return fmt.Errorf("confidence report: available status requires policy_version")
	}
	return nil
}

func validateFallbackConfidenceReport(a ConfidenceReportArtifact) error {
	if strings.TrimSpace(a.Reason) == "" {
		return fmt.Errorf("confidence report: %s status requires reason", a.Status)
	}
	return nil
}

func validateConfidenceReportCalibration(a ConfidenceReportArtifact) error {
	if err := ValidateCalibrationStatus(a.CalibrationStatus, a.SampleSize); err != nil {
		return fmt.Errorf("confidence report: %w", err)
	}
	if a.ConfidenceLevel != "" && a.ConfidenceLevel != ConfidenceLow && a.ConfidenceLevel != ConfidenceMedium && a.ConfidenceLevel != ConfidenceHigh {
		return fmt.Errorf("confidence report: unsupported confidence_level %q", a.ConfidenceLevel)
	}
	return nil
}

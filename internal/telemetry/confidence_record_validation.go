package telemetry

import (
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ValidateConfidenceRecord rejects malformed records before they can affect
// cross-agent comparisons or be mistaken for calibration evidence.
func ValidateConfidenceRecord(record ConfidenceRecord) error {
	if record.CoverageStatus == "" {
		record.CoverageStatus = ConfidenceCoverageReported
	}
	switch record.CoverageStatus {
	case ConfidenceCoverageMissing:
		return validateMissingConfidenceRecord(record)
	case ConfidenceCoverageRejected:
		return validateRejectedConfidenceRecord(record)
	case ConfidenceCoverageReported:
		return validateReportedConfidenceRecord(record)
	default:
		return fmt.Errorf("confidence record: coverage_status %q is not allowed", record.CoverageStatus)
	}
}

func validateMissingConfidenceRecord(record ConfidenceRecord) error {
	var errs []error
	if record.Agent == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires agent"))
	}
	if record.MissingReason == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires missing_reason"))
	}
	return errors.Join(errs...)
}

func validateRejectedConfidenceRecord(record ConfidenceRecord) error {
	var errs []error
	if record.ClaimID == "" {
		errs = append(errs, errors.New("confidence record: rejected observation requires claim_id"))
	}
	if record.Agent == "" {
		errs = append(errs, errors.New("confidence record: rejected observation requires agent"))
	}
	if record.Violation == "" {
		errs = append(errs, errors.New("confidence record: rejected observation requires violation"))
	}
	return errors.Join(errs...)
}

func validateReportedConfidenceRecord(record ConfidenceRecord) error {
	errs := validateReportedRecordIdentity(record)
	level, err := domain.ConfidenceLevelForPercent(record.ConfidencePercent)
	if err != nil {
		errs = append(errs, err)
	} else if record.ConfidenceLevel != level {
		errs = append(errs, fmt.Errorf("confidence record: confidence_level %q conflicts with derived %q", record.ConfidenceLevel, level))
	}
	errs = append(errs, validateRecordGroundTruth(record)...)
	errs = append(errs, validateRecordCalibration(record)...)
	return errors.Join(errs...)
}

func validateReportedRecordIdentity(record ConfidenceRecord) []error {
	errs := validateReportedRecordFields(record)
	return append(errs, validateRecordEvidenceClasses(record.EvidenceClasses)...)
}

func validateReportedRecordFields(record ConfidenceRecord) []error {
	var errs []error
	if record.ClaimID == "" {
		errs = append(errs, errors.New("confidence record: claim_id is required"))
	}
	if record.Agent == "" {
		errs = append(errs, errors.New("confidence record: agent is required"))
	}
	if record.ClaimKind != domain.ClaimKindQuestion && record.ClaimKind != domain.ClaimKindAssertion {
		errs = append(errs, fmt.Errorf("confidence record: claim_kind %q is not allowed", record.ClaimKind))
	}
	if record.ClaimKind == domain.ClaimKindAssertion && len(record.EvidenceIDs) == 0 && !record.EvidenceProvided {
		errs = append(errs, errors.New("confidence record: assertion requires evidence_ids or evidence_provided"))
	}
	return errs
}

func validateRecordEvidenceClasses(classes []string) []error {
	var errs []error
	for i, class := range classes {
		if class == "" {
			errs = append(errs, fmt.Errorf("confidence record: evidence_classes[%d] is empty", i))
		}
	}
	return errs
}

func validateRecordGroundTruth(record ConfidenceRecord) []error {
	errs := validateGroundTruthReference(record)
	return append(errs, validateRecordGroundTruthOutcome(record.GroundTruthRef, record.GroundTruthOutcome)...)
}

func validateGroundTruthReference(record ConfidenceRecord) []error {
	var errs []error
	if record.Corrected && record.GroundTruthRef == "" {
		errs = append(errs, errors.New("confidence record: corrected claim requires ground_truth_ref"))
	}
	if record.GroundTruthRef != "" && record.GroundTruthKind == "" {
		errs = append(errs, errors.New("confidence record: ground_truth_ref requires ground_truth_kind"))
	}
	if record.GroundTruthKind != "" && record.GroundTruthRef == "" {
		errs = append(errs, errors.New("confidence record: ground_truth_kind requires ground_truth_ref"))
	}
	return errs
}

func validateRecordGroundTruthOutcome(reference, outcome string) []error {
	if outcome == "" {
		return nil
	}
	var errs []error
	if reference == "" {
		errs = append(errs, errors.New("confidence record: ground_truth_outcome requires ground_truth_ref"))
	}
	if outcome != domain.GroundTruthCorrect && outcome != domain.GroundTruthIncorrect && outcome != domain.GroundTruthUnresolved {
		errs = append(errs, fmt.Errorf("confidence record: ground_truth_outcome %q is not allowed", outcome))
	}
	return errs
}

func validateRecordCalibration(record ConfidenceRecord) []error {
	errs := validateCalibrationClaim(record)
	if err := domain.ValidateCalibrationStatus(record.CalibrationStatus, record.SampleSize); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func validateCalibrationClaim(record ConfidenceRecord) []error {
	var errs []error
	if record.CalibrationStatus == domain.CalibrationCalibrated && (!record.Reviewed || record.SampleSize < domain.CalibrationMinimumSample) {
		errs = append(errs, fmt.Errorf("confidence record: calibrated requires reviewed sample_size >= %d", domain.CalibrationMinimumSample))
	}
	if record.GroundTruthKind != "" && record.GroundTruthKind != domain.GroundTruthUserRevision && record.GroundTruthKind != domain.GroundTruthHandoff && record.GroundTruthKind != domain.GroundTruthDownstream {
		errs = append(errs, fmt.Errorf("confidence record: ground_truth_kind %q is not allowed", record.GroundTruthKind))
	}
	if record.CalibrationStatus == domain.CalibrationCalibrated && record.GroundTruthRef == "" {
		errs = append(errs, errors.New("confidence record: calibrated claim requires ground_truth_ref"))
	}
	return errs
}

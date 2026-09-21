package telemetry

import (
	"errors"
	"fmt"
	"time"

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
	if record.MissionID == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires mission_id"))
	}
	if record.Agent == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires agent"))
	}
	if record.CorrelationKey == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires correlation_key"))
	}
	if record.MissingReason == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires missing_reason"))
	}
	if record.EventID == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires event_id"))
	}
	if record.Timestamp == "" {
		errs = append(errs, errors.New("confidence record: missing coverage requires timestamp"))
	} else if _, err := time.Parse(time.RFC3339, record.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("confidence record: missing coverage timestamp %q is not RFC3339", record.Timestamp))
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
	if record.CalibrationStatus != domain.CalibrationCalibrated {
		return validateGroundTruthKind(record.GroundTruthKind)
	}
	return append(validateCalibratedRecord(record), validateGroundTruthKind(record.GroundTruthKind)...)
}

func validateCalibratedRecord(record ConfidenceRecord) []error {
	var errs []error
	if !record.Reviewed || record.SampleSize < domain.CalibrationMinimumSample {
		errs = append(errs, fmt.Errorf("confidence record: calibrated requires reviewed sample_size >= %d", domain.CalibrationMinimumSample))
	}
	if record.GroundTruthRef == "" {
		errs = append(errs, errors.New("confidence record: calibrated claim requires ground_truth_ref"))
	}
	if record.GroundTruthOutcome != domain.GroundTruthCorrect && record.GroundTruthOutcome != domain.GroundTruthIncorrect {
		errs = append(errs, errors.New("confidence record: calibrated claim requires a resolved ground_truth_outcome"))
	}
	return errs
}

func validateGroundTruthKind(kind string) []error {
	if kind != "" && kind != domain.GroundTruthUserRevision && kind != domain.GroundTruthHandoff && kind != domain.GroundTruthDownstream {
		return []error{fmt.Errorf("confidence record: ground_truth_kind %q is not allowed", kind)}
	}
	return nil
}

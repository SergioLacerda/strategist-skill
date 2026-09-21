package domain

import (
	"errors"
	"fmt"
)

// Confidence policy v1 keeps the user-facing levels while making their
// percentage meaning explicit. Percentages are evidence-backed scores, not
// calibrated probabilities unless a separate calibration status says so.
const (
	ConfidencePolicyVersion = "v1"
	ConfidencePercentMin    = 0
	ConfidencePercentLowMax = 59
	ConfidencePercentMedMax = 84
	ConfidencePercentMax    = 100
)

// Claim kinds prevent an unresolved question from being rendered as a fact.
const (
	ClaimKindQuestion        = "question"
	ClaimKindAssertion       = "assertion"
	CalibrationNoSample      = "no_sample"
	CalibrationUncalibrated  = "uncalibrated"
	CalibrationObserved      = "observed"
	CalibrationCalibrated    = "calibrated"
	GroundTruthCorrect       = "correct"
	GroundTruthIncorrect     = "incorrect"
	GroundTruthUnresolved    = "unresolved"
	CalibrationMinimumSample = 3
	GroundTruthUserRevision  = "user_revision"
	GroundTruthHandoff       = "handoff_validation"
	GroundTruthDownstream    = "downstream_verification"
)

var allowedClaimKinds = stringSet(ClaimKindQuestion, ClaimKindAssertion)
var allowedCalibrationStatuses = stringSet(
	CalibrationNoSample,
	CalibrationUncalibrated,
	CalibrationObserved,
	CalibrationCalibrated,
)
var allowedGroundTruthKinds = stringSet(GroundTruthUserRevision, GroundTruthHandoff, GroundTruthDownstream)
var allowedGroundTruthOutcomes = stringSet(GroundTruthCorrect, GroundTruthIncorrect, GroundTruthUnresolved)

// ConfidenceClaim is the normalized contract shared by gate-relevant agent
// claims. Existing Decision and Evidence records remain backward compatible;
// new gate inputs should use this type.
type ConfidenceClaim struct {
	ID                 string   `yaml:"id" json:"id"`
	Statement          string   `yaml:"statement" json:"statement"`
	Agent              string   `yaml:"agent,omitempty" json:"agent,omitempty"`
	CorrelationKey     string   `yaml:"correlation_key,omitempty" json:"correlation_key,omitempty"`
	ClaimKind          string   `yaml:"claim_kind" json:"claim_kind"`
	ConfidencePercent  int      `yaml:"confidence_percent" json:"confidence_percent"`
	ConfidenceLevel    string   `yaml:"confidence_level,omitempty" json:"confidence_level,omitempty"`
	EvidenceIDs        []string `yaml:"evidence_ids,omitempty" json:"evidence_ids,omitempty"`
	EvidenceClasses    []string `yaml:"evidence_classes,omitempty" json:"evidence_classes,omitempty"`
	RiskLevel          string   `yaml:"risk_level,omitempty" json:"risk_level,omitempty"`
	Destructive        bool     `yaml:"destructive,omitempty" json:"destructive,omitempty"`
	Contradictions     []string `yaml:"contradictions,omitempty" json:"contradictions,omitempty"`
	CalibrationStatus  string   `yaml:"calibration_status,omitempty" json:"calibration_status,omitempty"`
	SampleSize         int      `yaml:"sample_size" json:"sample_size"`
	GroundTruthRef     string   `yaml:"ground_truth_ref,omitempty" json:"ground_truth_ref,omitempty"`
	GroundTruthKind    string   `yaml:"ground_truth_kind,omitempty" json:"ground_truth_kind,omitempty"`
	GroundTruthOutcome string   `yaml:"ground_truth_outcome,omitempty" json:"ground_truth_outcome,omitempty"`
}

// ConfidenceLevelForPercent derives the canonical level for a 0-100 score.
func ConfidenceLevelForPercent(percent int) (string, error) {
	if percent < ConfidencePercentMin || percent > ConfidencePercentMax {
		return "", fmt.Errorf("confidence_invalid: confidence_percent %d is out of range [0, 100]", percent)
	}
	switch {
	case percent <= ConfidencePercentLowMax:
		return ConfidenceLow, nil
	case percent <= ConfidencePercentMedMax:
		return ConfidenceMedium, nil
	default:
		return ConfidenceHigh, nil
	}
}

func validateConfidenceCompatibility(level string, percent *int) error {
	if percent == nil {
		return nil
	}
	derived, err := ConfidenceLevelForPercent(*percent)
	if err != nil {
		return err
	}
	if level != "" && level != derived {
		return fmt.Errorf("confidence_invalid: confidence %q conflicts with confidence_percent %d (derived %q)", level, *percent, derived)
	}
	return nil
}

// ValidateCalibrationStatus validates the common calibration vocabulary and
// makes an empty sample unambiguously non-calibrated.
func ValidateCalibrationStatus(status string, sampleSize int) error {
	if sampleSize < 0 {
		return errors.New("confidence_invalid: sample_size cannot be negative")
	}
	if sampleSize == 0 {
		return validateEmptyCalibrationStatus(status)
	}
	return validatePopulatedCalibrationStatus(status, sampleSize)
}

func validateEmptyCalibrationStatus(status string) error {
	if status != "" && status != CalibrationNoSample {
		return fmt.Errorf("confidence_invalid: sample_size 0 requires calibration_status %q", CalibrationNoSample)
	}
	return nil
}

func validatePopulatedCalibrationStatus(status string, sampleSize int) error {
	if status == CalibrationNoSample {
		return errors.New("confidence_invalid: no_sample requires sample_size 0")
	}
	if _, ok := allowedCalibrationStatuses[status]; !ok {
		return fmt.Errorf("confidence_invalid: calibration_status %q is not allowed", status)
	}
	return validateCalibratedSampleSize(status, sampleSize)
}

func validateCalibratedSampleSize(status string, sampleSize int) error {
	if status == CalibrationCalibrated && sampleSize < CalibrationMinimumSample {
		return fmt.Errorf("confidence_invalid: calibrated requires at least %d reviewed samples", CalibrationMinimumSample)
	}
	return nil
}

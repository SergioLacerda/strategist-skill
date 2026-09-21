package domain

import (
	"errors"
	"fmt"
)

// ConfidenceSummary is the handoff-safe envelope. MissingRecord is explicit so
// legacy or unavailable producers cannot be mistaken for an empty successful
// summary.
type ConfidenceSummary struct {
	PolicyVersion     string            `yaml:"policy_version" json:"policy_version"`
	Claims            []ConfidenceClaim `yaml:"claims" json:"claims"`
	OpenQuestions     []ConfidenceClaim `yaml:"open_questions,omitempty" json:"open_questions,omitempty"`
	Evidence          []Evidence        `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	SampleSize        int               `yaml:"sample_size" json:"sample_size"`
	CalibrationStatus string            `yaml:"calibration_status" json:"calibration_status"`
	MissingRecord     bool              `yaml:"missing_record" json:"missing_record"`
	Violations        []string          `yaml:"violations,omitempty" json:"violations,omitempty"`
}

// ValidateConfidenceSummary validates all claims before a handoff can expose
// them to the Approval Gate.
func ValidateConfidenceSummary(summary ConfidenceSummary) error {
	errs := validateConfidenceSummaryHeader(summary)
	if summary.MissingRecord {
		errs = append(errs, validateMissingConfidenceSummary(summary)...)
		return errors.Join(errs...)
	}
	if len(summary.Claims) == 0 && len(summary.OpenQuestions) == 0 {
		errs = append(errs, errors.New("confidence_summary_invalid: empty summary requires missing_record"))
	}
	errs = append(errs, validateSummaryEvidence(summary.Evidence)...)
	errs = append(errs, validateSummaryClaims(summary)...)
	errs = append(errs, validateSummaryCalibration(summary)...)
	return errors.Join(errs...)
}

// CompareConfidenceSummaries proves that claims survive a handoff by stable
// correlation key. Destination summaries may add claims, but cannot silently
// drop or rewrite a source claim.
func CompareConfidenceSummaries(source, destination ConfidenceSummary) error {
	if err := ValidateConfidenceSummary(source); err != nil {
		return fmt.Errorf("source confidence summary: %w", err)
	}
	if err := ValidateConfidenceSummary(destination); err != nil {
		return fmt.Errorf("destination confidence summary: %w", err)
	}
	if source.MissingRecord {
		return nil
	}
	destinationByCorrelation := confidenceClaimsByCorrelation(destination)
	return compareConfidenceClaims(source, destinationByCorrelation)
}

func confidenceClaimsByCorrelation(summary ConfidenceSummary) map[string]ConfidenceClaim {
	claims := append(append([]ConfidenceClaim(nil), summary.Claims...), summary.OpenQuestions...)
	byCorrelation := make(map[string]ConfidenceClaim, len(claims))
	for _, claim := range claims {
		byCorrelation[claim.CorrelationKey] = claim
	}
	return byCorrelation
}

func compareConfidenceClaims(source ConfidenceSummary, destinationByCorrelation map[string]ConfidenceClaim) error {
	var errs []error
	sourceClaims := append(append([]ConfidenceClaim(nil), source.Claims...), source.OpenQuestions...)
	for _, sourceClaim := range sourceClaims {
		destinationClaim, ok := destinationByCorrelation[sourceClaim.CorrelationKey]
		if !ok {
			errs = append(errs, fmt.Errorf("confidence_handoff_invalid: claim %q correlation %q was not preserved", sourceClaim.ID, sourceClaim.CorrelationKey))
			continue
		}
		errs = append(errs, compareConfidenceClaim(sourceClaim, destinationClaim)...)
	}
	return errors.Join(errs...)
}

func compareConfidenceClaim(source, destination ConfidenceClaim) []error {
	var errs []error
	errs = append(errs, compareClaimIdentity(source, destination)...)
	errs = append(errs, compareClaimEvidence(source, destination)...)
	errs = append(errs, compareClaimGroundTruth(source, destination)...)
	errs = append(errs, compareClaimCalibration(source, destination)...)
	return errs
}

func compareClaimIdentity(source, destination ConfidenceClaim) []error {
	var errs []error
	if source.Statement != destination.Statement {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed statement", source.CorrelationKey))
	}
	if source.ClaimKind != destination.ClaimKind {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed claim_kind from %q to %q", source.CorrelationKey, source.ClaimKind, destination.ClaimKind))
	}
	if source.ConfidencePercent != destination.ConfidencePercent || source.ConfidenceLevel != destination.ConfidenceLevel {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed confidence", source.CorrelationKey))
	}
	return errs
}

func compareClaimEvidence(source, destination ConfidenceClaim) []error {
	var errs []error
	if !sameStrings(source.EvidenceIDs, destination.EvidenceIDs) {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed evidence_ids", source.CorrelationKey))
	}
	if !sameStrings(source.EvidenceClasses, destination.EvidenceClasses) {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed evidence_classes", source.CorrelationKey))
	}
	return errs
}

func compareClaimGroundTruth(source, destination ConfidenceClaim) []error {
	if changedGroundTruth(source, destination) {
		return []error{fmt.Errorf("confidence_handoff_invalid: correlation %q changed ground truth", source.CorrelationKey)}
	}
	return nil
}

func compareClaimCalibration(source, destination ConfidenceClaim) []error {
	if source.CalibrationStatus != destination.CalibrationStatus || source.SampleSize != destination.SampleSize {
		return []error{fmt.Errorf("confidence_handoff_invalid: correlation %q changed calibration", source.CorrelationKey)}
	}
	return nil
}

func changedGroundTruth(source, destination ConfidenceClaim) bool {
	return source.GroundTruthRef != destination.GroundTruthRef ||
		source.GroundTruthKind != destination.GroundTruthKind ||
		source.GroundTruthOutcome != destination.GroundTruthOutcome
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

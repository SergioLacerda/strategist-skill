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
	return errors.Join(errs...)
}
func validateConfidenceSummaryHeader(summary ConfidenceSummary) []error {
	var errs []error
	if summary.PolicyVersion != ConfidencePolicyVersion {
		errs = append(errs, fmt.Errorf("confidence_summary_invalid: policy_version %q is not %q", summary.PolicyVersion, ConfidencePolicyVersion))
	}
	if summary.SampleSize < 0 {
		errs = append(errs, errors.New("confidence_summary_invalid: sample_size cannot be negative"))
	}
	if err := ValidateCalibrationStatus(summary.CalibrationStatus, summary.SampleSize); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func validateMissingConfidenceSummary(summary ConfidenceSummary) []error {
	var errs []error
	if len(summary.Claims) != 0 || len(summary.OpenQuestions) != 0 || len(summary.Evidence) != 0 {
		errs = append(errs, errors.New("confidence_summary_invalid: missing_record cannot carry claims or evidence"))
	}
	if summary.SampleSize != 0 || summary.CalibrationStatus != CalibrationNoSample {
		errs = append(errs, errors.New("confidence_summary_invalid: missing_record requires sample_size 0 and no_sample"))
	}
	return errs
}

func validateSummaryEvidence(evidence []Evidence) []error {
	var errs []error
	seen := make(map[string]struct{}, len(evidence))
	for _, item := range evidence {
		if _, exists := seen[item.ID]; exists {
			errs = append(errs, fmt.Errorf("confidence_summary_invalid: duplicate evidence id %q", item.ID))
		}
		seen[item.ID] = struct{}{}
		if err := ValidateEvidence(item); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateSummaryClaims(summary ConfidenceSummary) []error {
	claims := append(append([]ConfidenceClaim(nil), summary.Claims...), summary.OpenQuestions...)
	openQuestions := make(map[string]struct{}, len(summary.OpenQuestions))
	for _, claim := range summary.OpenQuestions {
		openQuestions[claim.ID] = struct{}{}
	}
	seenClaims := make(map[string]struct{}, len(claims))
	var errs []error
	for _, claim := range claims {
		if _, exists := seenClaims[claim.ID]; exists {
			errs = append(errs, fmt.Errorf("confidence_summary_invalid: duplicate claim id %q", claim.ID))
		}
		seenClaims[claim.ID] = struct{}{}
		errs = append(errs, validateSummaryClaim(claim, summary.Evidence, openQuestions)...)
	}
	return errs
}

func validateSummaryClaim(claim ConfidenceClaim, evidence []Evidence, openQuestions map[string]struct{}) []error {
	var errs []error
	if claim.Agent == "" {
		errs = append(errs, fmt.Errorf("confidence_summary_invalid: claim %q requires agent", claim.ID))
	}
	if claim.CorrelationKey == "" {
		errs = append(errs, fmt.Errorf("confidence_summary_invalid: claim %q requires correlation_key", claim.ID))
	}
	if _, isOpenQuestion := openQuestions[claim.ID]; isOpenQuestion && claim.ClaimKind != ClaimKindQuestion {
		errs = append(errs, fmt.Errorf("confidence_summary_invalid: open question %q must have claim_kind question", claim.ID))
	}
	if err := ValidateConfidenceClaim(claim, evidence); err != nil {
		errs = append(errs, err)
	}
	return errs
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
	if source.ClaimKind != destination.ClaimKind {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed claim_kind from %q to %q", source.CorrelationKey, source.ClaimKind, destination.ClaimKind))
	}
	if source.ConfidencePercent != destination.ConfidencePercent || source.ConfidenceLevel != destination.ConfidenceLevel {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed confidence", source.CorrelationKey))
	}
	if !sameStrings(source.EvidenceIDs, destination.EvidenceIDs) {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed evidence_ids", source.CorrelationKey))
	}
	if !sameStrings(source.EvidenceClasses, destination.EvidenceClasses) {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed evidence_classes", source.CorrelationKey))
	}
	if changedGroundTruth(source, destination) {
		errs = append(errs, fmt.Errorf("confidence_handoff_invalid: correlation %q changed ground truth", source.CorrelationKey))
	}
	return errs
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

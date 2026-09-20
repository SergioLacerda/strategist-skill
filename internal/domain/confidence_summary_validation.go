package domain

import (
	"errors"
	"fmt"
)

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

func validateSummaryCalibration(summary ConfidenceSummary) []error {
	if summary.CalibrationStatus != CalibrationCalibrated {
		return nil
	}
	claims := append(append([]ConfidenceClaim(nil), summary.Claims...), summary.OpenQuestions...)
	for _, claim := range claims {
		if claim.GroundTruthRef != "" && (claim.GroundTruthOutcome == GroundTruthCorrect || claim.GroundTruthOutcome == GroundTruthIncorrect) {
			return nil
		}
	}
	return []error{errors.New("confidence_summary_invalid: calibrated summary requires a claim with resolved ground truth")}
}

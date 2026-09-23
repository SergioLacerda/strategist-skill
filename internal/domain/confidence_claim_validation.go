package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateConfidenceClaim enforces the fail-closed language/evidence rules at
// the boundary where a claim becomes gate-visible.
func ValidateConfidenceClaim(claim ConfidenceClaim, evidence []Evidence) error {
	level, errs := validateConfidenceClaimFields(claim)
	if len(claim.EvidenceClasses) > 0 && len(claim.EvidenceClasses) != len(claim.EvidenceIDs) {
		errs = append(errs, errors.New("confidence_invalid: evidence_classes must align with evidence_ids"))
	}
	if claim.ClaimKind == ClaimKindAssertion {
		errs = append(errs, validateAssertionClaim(level, claim, evidence)...)
	}
	return errors.Join(errs...)
}

func validateConfidenceClaimFields(claim ConfidenceClaim) (string, []error) {
	var errs []error
	if claim.ID == "" {
		errs = append(errs, errors.New("confidence_invalid: id is required"))
	}
	if claim.Statement == "" {
		errs = append(errs, errors.New("confidence_invalid: statement is required"))
	}
	if _, ok := allowedClaimKinds[claim.ClaimKind]; !ok {
		errs = append(errs, fmt.Errorf("confidence_invalid: claim_kind %q is not allowed", claim.ClaimKind))
	}
	level, err := ConfidenceLevelForPercent(claim.ConfidencePercent)
	if err != nil {
		errs = append(errs, err)
	} else if claim.ConfidenceLevel != "" && claim.ConfidenceLevel != level {
		errs = append(errs, fmt.Errorf("confidence_invalid: confidence_level %q conflicts with derived %q", claim.ConfidenceLevel, level))
	}
	if err := ValidateCalibrationStatus(claim.CalibrationStatus, claim.SampleSize); err != nil {
		errs = append(errs, err)
	}
	errs = append(errs, validateGroundTruthFields(claim)...)
	errs = append(errs, validateClaimCalibration(claim)...)
	return level, errs
}

func validateClaimCalibration(claim ConfidenceClaim) []error {
	if claim.CalibrationStatus != CalibrationCalibrated {
		return nil
	}
	var errs []error
	if claim.GroundTruthRef == "" {
		errs = append(errs, errors.New("confidence_invalid: calibrated claim requires ground_truth_ref"))
	}
	if claim.GroundTruthKind == "" {
		errs = append(errs, errors.New("confidence_invalid: calibrated claim requires ground_truth_kind"))
	}
	if claim.GroundTruthOutcome != GroundTruthCorrect && claim.GroundTruthOutcome != GroundTruthIncorrect {
		errs = append(errs, errors.New("confidence_invalid: calibrated claim requires a resolved outcome"))
	}
	return errs
}

func validateGroundTruthFields(claim ConfidenceClaim) []error {
	var errs []error
	if claim.GroundTruthRef != "" && claim.GroundTruthKind == "" {
		errs = append(errs, errors.New("confidence_invalid: ground_truth_ref requires ground_truth_kind"))
	}
	errs = append(errs, validateGroundTruthKind(claim.GroundTruthKind)...)
	errs = append(errs, validateGroundTruthOutcome(claim.GroundTruthRef, claim.GroundTruthOutcome)...)
	return errs
}

func validateGroundTruthKind(kind string) []error {
	if kind != "" {
		if _, ok := allowedGroundTruthKinds[kind]; !ok {
			return []error{fmt.Errorf("confidence_invalid: ground_truth_kind %q is not allowed", kind)}
		}
	}
	return nil
}

func validateGroundTruthOutcome(reference, outcome string) []error {
	if outcome == "" {
		return nil
	}
	var errs []error
	if _, ok := allowedGroundTruthOutcomes[outcome]; !ok {
		errs = append(errs, fmt.Errorf("confidence_invalid: ground_truth_outcome %q is not allowed", outcome))
	}
	if reference == "" {
		errs = append(errs, errors.New("confidence_invalid: ground_truth_outcome requires ground_truth_ref"))
	}
	return errs
}

func validateAssertionClaim(level string, claim ConfidenceClaim, evidence []Evidence) []error {
	if len(claim.EvidenceIDs) == 0 {
		return []error{errors.New("confidence_invalid: assertion requires evidence")}
	}
	evidenceByID := make(map[string]Evidence, len(evidence))
	for _, item := range evidence {
		evidenceByID[item.ID] = item
	}
	resolved, errs := resolveAssertionEvidence(claim.EvidenceIDs, evidenceByID)
	errs = append(errs, validateEvidenceClasses(claim.EvidenceClasses)...)
	return append(errs, validateAssertionRules(level, claim, resolved)...)
}

func resolveAssertionEvidence(ids []string, evidenceByID map[string]Evidence) ([]Evidence, []error) {
	resolved := make([]Evidence, 0, len(ids))
	var errs []error
	for _, id := range ids {
		item, ok := evidenceByID[id]
		if !ok {
			errs = append(errs, fmt.Errorf("confidence_invalid: assertion references unresolved evidence %q", id))
			continue
		}
		resolved = append(resolved, item)
	}
	return resolved, errs
}

func validateEvidenceClasses(classes []string) []error {
	var errs []error
	for i, class := range classes {
		if _, ok := allowedEvidenceClasses[class]; !ok {
			errs = append(errs, fmt.Errorf("confidence_invalid: evidence_classes[%d] %q is not allowed (want one of %s)", i, class, strings.Join(orderedEvidenceClasses, ", ")))
		}
	}
	return errs
}

func validateAssertionRules(level string, claim ConfidenceClaim, evidence []Evidence) []error {
	var errs []error
	if level == ConfidenceLow {
		errs = append(errs, errors.New("confidence_invalid: low-confidence claims must be questions"))
	}
	for _, item := range evidence {
		if item.Class != EvidenceClassExplicit && item.Class != EvidenceClassCorroboratedInference {
			errs = append(errs, fmt.Errorf("confidence_invalid: assertion evidence %q has non-supporting class %q", item.ID, item.Class))
		}
	}
	errs = append(errs, validateMediumAssertion(level, claim)...)
	if level == ConfidenceHigh {
		errs = append(errs, validateHighAssertion(claim, evidence)...)
	}
	return errs
}

func validateMediumAssertion(level string, claim ConfidenceClaim) []error {
	if level == ConfidenceMedium && (claim.Destructive || claim.RiskLevel == "high") {
		return []error{errors.New("confidence_invalid: medium-confidence assertions cannot authorize high-risk or destructive work")}
	}
	return nil
}

func validateHighAssertion(claim ConfidenceClaim, evidence []Evidence) []error {
	var errs []error
	if len(claim.Contradictions) > 0 {
		errs = append(errs, errors.New("confidence_invalid: high-confidence assertion has unresolved contradictions"))
	}
	explicit, sourceCount := highAssertionEvidence(evidence)
	if !explicit && sourceCount < 2 {
		errs = append(errs, errors.New("confidence_invalid: high-confidence assertion requires explicit evidence or two independent corroborated sources"))
	}
	return errs
}

func highAssertionEvidence(evidence []Evidence) (bool, int) {
	explicit := false
	sources := make(map[string]struct{})
	for _, item := range evidence {
		if item.Class == EvidenceClassExplicit {
			explicit = true
		}
		if item.Class == EvidenceClassCorroboratedInference {
			sources[item.SourceRef] = struct{}{}
		}
	}
	return explicit, len(sources)
}

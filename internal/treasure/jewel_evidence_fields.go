package treasure

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// validateJewelEvidenceFields checks the additive evidence_class/
// evidence_confidence/valid_until fields: each is independently optional,
// but when set must be a value domain.Evidence's own vocabulary (or, for
// valid_until, RFC3339) already recognizes.
func validateJewelEvidenceFields(j Jewel) error {
	if err := validateJewelEvidenceClass(j); err != nil {
		return err
	}
	if err := validateJewelEvidenceConfidence(j); err != nil {
		return err
	}
	return validateJewelValidUntil(j)
}

func validateJewelEvidenceClass(j Jewel) error {
	if j.EvidenceClass == "" {
		return nil
	}
	if _, ok := allowedJewelEvidenceClasses[j.EvidenceClass]; !ok {
		return fmt.Errorf("jewel %q has invalid evidence_class %q", j.ID, j.EvidenceClass)
	}
	return nil
}

func validateJewelEvidenceConfidence(j Jewel) error {
	if j.EvidenceConfidence == "" {
		return nil
	}
	if _, ok := allowedJewelEvidenceConfidences[j.EvidenceConfidence]; !ok {
		return fmt.Errorf("jewel %q has invalid evidence_confidence %q", j.ID, j.EvidenceConfidence)
	}
	return nil
}

func validateJewelValidUntil(j Jewel) error {
	if j.ValidUntil == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, j.ValidUntil); err != nil {
		return fmt.Errorf("jewel %q has invalid valid_until %q: not RFC3339", j.ID, j.ValidUntil)
	}
	return nil
}

var allowedJewelEvidenceClasses = stringSet(
	domain.EvidenceClassExplicit,
	domain.EvidenceClassCorroboratedInference,
	domain.EvidenceClassWeakInference,
	domain.EvidenceClassUnknown,
)

var allowedJewelEvidenceConfidences = stringSet(
	domain.ConfidenceLow,
	domain.ConfidenceMedium,
	domain.ConfidenceHigh,
)

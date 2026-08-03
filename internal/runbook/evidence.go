package runbook

import (
	"errors"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EvidenceRef is a runbook step's reference to a graded, sourced claim,
// reusing domain.Evidence's Class/Confidence vocabulary rather than
// reinventing it — the same reuse-not-reinvent pattern the sibling
// jewel-evidence-wiring and handoff-challenge-extensions missions used, per
// design.md § internal/runbook Package.
type EvidenceRef struct {
	ID         string `yaml:"id"`
	Class      string `yaml:"class"`
	Confidence string `yaml:"confidence"`
}

// ValidateEvidenceRef checks an EvidenceRef against domain's own Evidence
// class/confidence vocabulary.
func ValidateEvidenceRef(e EvidenceRef) error {
	var errs []error
	errs = append(errs, validateNamedValue("evidence_ref_invalid", "id", e.ID, nil)...)
	errs = append(errs, validateDomainEvidenceClass(e.Class)...)
	errs = append(errs, validateDomainConfidence(e.Confidence)...)
	return errors.Join(errs...)
}

// qualifies reports whether the evidence's Class is strong enough to count
// as satisfying a check on its own, without a caller-supplied justification
// — explicit and corroborated_inference qualify; weak_inference and unknown
// do not, per domain.Evidence's own "never promote an inference to a fact"
// stance.
func (e EvidenceRef) qualifies() bool {
	return e.Class == domain.EvidenceClassExplicit || e.Class == domain.EvidenceClassCorroboratedInference
}

func validateDomainEvidenceClass(class string) []error {
	allowed := stringSet(
		domain.EvidenceClassExplicit,
		domain.EvidenceClassCorroboratedInference,
		domain.EvidenceClassWeakInference,
		domain.EvidenceClassUnknown,
	)
	return validateNamedValue("evidence_ref_invalid", "class", class, allowed)
}

func validateDomainConfidence(confidence string) []error {
	allowed := stringSet(domain.ConfidenceLow, domain.ConfidenceMedium, domain.ConfidenceHigh)
	return validateNamedValue("evidence_ref_invalid", "confidence", confidence, allowed)
}

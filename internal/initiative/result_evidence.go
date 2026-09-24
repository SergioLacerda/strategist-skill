package initiative

import (
	"fmt"
	"strings"
)

func validateEvidence(result Result) error {
	global, err := indexEvidence(result.EvidenceRefs)
	if err != nil {
		return err
	}
	return validateEvidenceLinks(result, global)
}

// indexEvidence validates every top-level evidence ref and indexes it by id.
func indexEvidence(refs []EvidenceRef) (map[string]EvidenceRef, error) {
	global := make(map[string]EvidenceRef, len(refs))
	for _, ref := range refs {
		if err := validateEvidenceRef(ref); err != nil {
			return nil, err
		}
		if _, exists := global[ref.ID]; exists {
			return nil, fmt.Errorf("initiative_evidence_invalid: duplicate evidence %q", ref.ID)
		}
		global[ref.ID] = ref
	}
	return global, nil
}

// validateEvidenceLinks requires every check and outcome ref to point at a
// top-level evidence ref.
func validateEvidenceLinks(result Result, global map[string]EvidenceRef) error {
	for _, check := range result.Checks {
		if err := validateCheckEvidence(check, global); err != nil {
			return err
		}
	}
	for _, outcome := range result.Outcomes {
		if err := validateLinkedRefs("outcome", outcome.ID, outcome.EvidenceRefs, global); err != nil {
			return err
		}
	}
	return nil
}

func validateCheckEvidence(check ObligationCheck, global map[string]EvidenceRef) error {
	seen := make(map[string]struct{}, len(check.EvidenceRefs))
	for _, ref := range check.EvidenceRefs {
		if err := validateUniqueCheckRef(ref, seen); err != nil {
			return err
		}
		if err := requireLinked("check", check.ID, ref, global); err != nil {
			return err
		}
	}
	return nil
}

func validateUniqueCheckRef(ref EvidenceRef, seen map[string]struct{}) error {
	if err := validateEvidenceRef(ref); err != nil {
		return err
	}
	if _, exists := seen[ref.ID]; exists {
		return fmt.Errorf("initiative_evidence_invalid: duplicate check evidence %q", ref.ID)
	}
	seen[ref.ID] = struct{}{}
	return nil
}

func validateLinkedRefs(kind, owner string, refs []EvidenceRef, global map[string]EvidenceRef) error {
	for _, ref := range refs {
		if err := validateEvidenceRef(ref); err != nil {
			return err
		}
		if err := requireLinked(kind, owner, ref, global); err != nil {
			return err
		}
	}
	return nil
}

func requireLinked(kind, owner string, ref EvidenceRef, global map[string]EvidenceRef) error {
	if _, exists := global[ref.ID]; !exists {
		return fmt.Errorf("initiative_evidence_invalid: %s %q references unlinked evidence %q", kind, owner, ref.ID)
	}
	return nil
}

func validateEvidenceRef(ref EvidenceRef) error {
	if strings.TrimSpace(ref.ID) == "" {
		return fmt.Errorf("initiative_evidence_invalid: evidence id is required")
	}
	if len(ref.ID) > maxInitiativeFieldBytes || len(ref.Fingerprint) > maxInitiativeFieldBytes {
		return fmt.Errorf("initiative_evidence_invalid: evidence %q exceeds the field size limit", ref.ID)
	}
	if _, ok := validEvidenceClasses[ref.Class]; !ok {
		return fmt.Errorf("initiative_evidence_invalid: evidence %q has invalid class %q", ref.ID, ref.Class)
	}
	if ref.Fingerprint != "" && len(strings.TrimSpace(ref.Fingerprint)) < 16 {
		return fmt.Errorf("initiative_evidence_invalid: evidence %q has an invalid fingerprint", ref.ID)
	}
	return nil
}

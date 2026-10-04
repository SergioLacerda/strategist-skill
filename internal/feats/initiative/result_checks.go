package initiative

import (
	"fmt"
	"strings"
)

func validateChecks(checks []ObligationCheck, expected []string) error {
	expectedSet := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		expectedSet[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		if err := validateExpectedCheck(check, seen, expectedSet); err != nil {
			return err
		}
		seen[check.ID] = struct{}{}
	}
	for _, id := range expected {
		if _, exists := seen[id]; !exists {
			return fmt.Errorf("initiative_obligation_invalid: missing check %q", id)
		}
	}
	return nil
}

func validateExpectedCheck(check ObligationCheck, seen, expectedSet map[string]struct{}) error {
	if err := validateCheck(check); err != nil {
		return err
	}
	if _, exists := seen[check.ID]; exists {
		return fmt.Errorf("initiative_obligation_invalid: duplicate check %q", check.ID)
	}
	if _, exists := expectedSet[check.ID]; !exists {
		return fmt.Errorf("initiative_obligation_invalid: unknown check %q", check.ID)
	}
	return nil
}

func validateCheck(check ObligationCheck) error {
	if strings.TrimSpace(check.ID) == "" || !validCheckStatus(check.Status) {
		return fmt.Errorf("initiative_result_invalid: check id and valid status are required")
	}
	if len(check.ID) > maxInitiativeFieldBytes || len(check.Reason) > maxInitiativeFieldBytes {
		return fmt.Errorf("initiative_result_invalid: check %q exceeds the field size limit", check.ID)
	}
	return validateCheckStatusRequirements(check)
}

func validateCheckStatusRequirements(check ObligationCheck) error {
	if check.Status == CheckSatisfied && len(check.EvidenceRefs) == 0 {
		return fmt.Errorf("initiative_result_invalid: satisfied check %q requires evidence", check.ID)
	}
	if check.Status == CheckNotApplicable && strings.TrimSpace(check.Reason) == "" {
		return fmt.Errorf("initiative_obligation_invalid: not_applicable check %q requires a reason", check.ID)
	}
	return nil
}

func validCheckStatus(status CheckStatus) bool {
	switch status {
	case CheckSatisfied, CheckPartial, CheckBlocked, CheckNotApplicable:
		return true
	default:
		return false
	}
}

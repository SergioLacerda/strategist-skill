package eval

import (
	"fmt"
	"strings"
)

func validateChestGrade(_ string, sourceGrade, reuseValue, implementationStatus string) error {
	if sourceGrade != "" && !contains([]string{"A", "B", "C"}, sourceGrade) {
		return fmt.Errorf("source_grade %q must be one of A, B, C", sourceGrade)
	}
	if reuseValue != "" && !contains([]string{"high", "medium", "low"}, reuseValue) {
		return fmt.Errorf("reuse_value %q must be one of high, medium, low", reuseValue)
	}
	if implementationStatus != "" && !contains([]string{"not_started", "in_progress", "implemented", "deprecated"}, implementationStatus) {
		return fmt.Errorf("implementation_status %q must be one of not_started, in_progress, implemented, deprecated", implementationStatus)
	}
	return nil
}

func validateJewelTrust(jewelID, jewelTrust, chestTrustTier string) error {
	order := map[string]int{"T0": 0, "T1": 1, "T2": 2, "T3": 3}
	jt, ok := order[jewelTrust]
	if !ok {
		return fmt.Errorf("jewel %q trust %q must be one of T0, T1, T2, T3", jewelID, jewelTrust)
	}
	ct, ok := order[chestTrustTier]
	if !ok {
		return fmt.Errorf("jewel %q parent chest trust tier %q must be one of T0, T1, T2, T3", jewelID, chestTrustTier)
	}
	if jt < ct {
		return fmt.Errorf("jewel %q trust %q exceeds parent chest's trust tier %q", jewelID, jewelTrust, chestTrustTier)
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

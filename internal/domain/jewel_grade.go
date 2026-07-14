package domain

import "fmt"

// tierOrder ranks trust tiers from most trusted (0) to least trusted (3).
var tierOrder = map[string]int{"T0": 0, "T1": 1, "T2": 2, "T3": 3}

// ValidateJewelTrust returns an error if jewelTrust is more trusted than chestTrustTier —
// a jewel can never exceed the trust tier of the chest it was generated from. This is the
// safeguard that replaces a human pre-approval gate for agent-generated jewels (see
// .analysis/refined/bau-tesouro-sq003-004-007/design.md).
func ValidateJewelTrust(jewelID, jewelTrust, chestTrustTier string) error {
	jt, ok := tierOrder[jewelTrust]
	if !ok {
		return fmt.Errorf("jewel %q trust %q must be one of T0, T1, T2, T3", jewelID, jewelTrust)
	}
	ct, ok := tierOrder[chestTrustTier]
	if !ok {
		return fmt.Errorf("jewel %q parent chest trust tier %q must be one of T0, T1, T2, T3", jewelID, chestTrustTier)
	}
	if jt < ct {
		return fmt.Errorf("jewel %q trust %q exceeds parent chest's trust tier %q", jewelID, jewelTrust, chestTrustTier)
	}
	return nil
}

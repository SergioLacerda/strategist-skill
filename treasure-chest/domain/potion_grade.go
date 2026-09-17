package domain

import "fmt"

// Potion lifecycle statuses mirror Jewel's (proposed, accepted, verified, deprecated) —
// Potion never carried the legacy "active" status Jewel had to migrate away from (see
// ADR 0012), so there is no legacy-status special case here.
const (
	PotionStatusProposed   = "proposed"
	PotionStatusAccepted   = "accepted"
	PotionStatusVerified   = "verified"
	PotionStatusDeprecated = "deprecated"
)

var validPotionStatuses = map[string]bool{
	PotionStatusProposed:   true,
	PotionStatusAccepted:   true,
	PotionStatusVerified:   true,
	PotionStatusDeprecated: true,
}

// ValidatePotionStatus returns an error if status is not one of the four lifecycle states.
func ValidatePotionStatus(potionID, status string) error {
	if validPotionStatuses[status] {
		return nil
	}
	return fmt.Errorf("potion %q status %q must be one of proposed, accepted, verified, deprecated", potionID, status)
}

// ValidatePotionTrust returns an error if potionTrust is more trusted than chestTrustTier —
// same doctrine as ValidateJewelTrust: a potion can never exceed its parent chest's tier.
func ValidatePotionTrust(potionID, potionTrust, chestTrustTier string) error {
	pt, ok := tierOrder[potionTrust]
	if !ok {
		return fmt.Errorf("potion %q trust %q must be one of T0, T1, T2, T3", potionID, potionTrust)
	}
	ct, ok := tierOrder[chestTrustTier]
	if !ok {
		return fmt.Errorf("potion %q parent chest trust tier %q must be one of T0, T1, T2, T3", potionID, chestTrustTier)
	}
	if pt < ct {
		return fmt.Errorf("potion %q trust %q exceeds parent chest's trust tier %q", potionID, potionTrust, chestTrustTier)
	}
	return nil
}

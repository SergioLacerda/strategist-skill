package domain

import "fmt"

// tierOrder ranks trust tiers from most trusted (0) to least trusted (3).
var tierOrder = map[string]int{"T0": 0, "T1": 1, "T2": 2, "T3": 3}

// Jewel lifecycle statuses (see ADR 0012). "active" was the pre-migration status and is no
// longer valid — ValidateJewelStatus rejects it explicitly so migration drift fails loudly.
const (
	JewelStatusProposed   = "proposed"
	JewelStatusAccepted   = "accepted"
	JewelStatusVerified   = "verified"
	JewelStatusDeprecated = "deprecated"

	// jewelStatusLegacyActive is no longer a valid status. It is recognized only so
	// ValidateJewelStatus can return a migration-specific error message instead of a
	// generic "not a valid status" one.
	jewelStatusLegacyActive = "active"
)

var validJewelStatuses = map[string]bool{
	JewelStatusProposed:   true,
	JewelStatusAccepted:   true,
	JewelStatusVerified:   true,
	JewelStatusDeprecated: true,
}

// ValidateJewelStatus returns an error if status is not one of the four lifecycle states
// (proposed, accepted, verified, deprecated). The legacy "active" status is called out by
// name since it was the pre-migration default and existing jewels.yaml files may still
// carry it — see ADR 0012 and `strategist treasure-chest mine --migrate-status`.
func ValidateJewelStatus(jewelID, status string) error {
	if validJewelStatuses[status] {
		return nil
	}
	if status == jewelStatusLegacyActive {
		return fmt.Errorf(
			"jewel %q has legacy status %q, no longer valid; run `strategist treasure-chest mine --migrate-status` to migrate active -> accepted",
			jewelID, status,
		)
	}
	return fmt.Errorf("jewel %q status %q must be one of proposed, accepted, verified, deprecated", jewelID, status)
}

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

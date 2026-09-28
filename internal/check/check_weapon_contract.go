package check

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateWeaponBoundary(slot, roleID string, contract domain.WeaponContract) error {
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("weapon contract invalid: %w", err)
	}
	if slot != string(domain.SlotDiscovery) || contract.IsZero() {
		return nil
	}
	if contract.RoleOwner != roleID {
		return fmt.Errorf("discovery weapon contract role_owner=%q does not match role %q", contract.RoleOwner, roleID)
	}
	return nil
}

// validateSelectedDiscoveryWeaponContract applies the stronger boundary only
// to the provider actually selected for discovery. Optional catalogued Ranger
// candidates may omit the contract until selected; the selected Weapon may not.
func validateSelectedDiscoveryWeaponContract(root, provider string) error {
	facts, err := domain.ResolveWeaponFacts(root, provider)
	if err != nil {
		return fmt.Errorf("discovery weapon %q unavailable: %w", provider, err)
	}
	if facts.CanonicalRole != "ranger" {
		return fmt.Errorf("discovery weapon %q is incompatible with Ranger", provider)
	}
	if facts.WeaponContract.IsZero() {
		return fmt.Errorf("discovery weapon contract missing: invocation evidence and fail-closed behavior are required")
	}
	return validateWeaponBoundary(string(domain.SlotDiscovery), "ranger", facts.WeaponContract)
}

// weaponBindingErrors renders every failed binding as a check.go-style error
// string, for inclusion in the same errs slice every other check.go
// validation gates the command's exit code on.
func weaponBindingErrors(bindings []weaponBinding) []string {
	var errs []string
	for _, b := range bindings {
		if !b.OK {
			errs = append(errs, fmt.Sprintf("weapon binding %s: %s", b.SkillID, b.Reason))
		}
	}
	return errs
}

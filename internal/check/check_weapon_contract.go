package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
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
	path := filepath.Join(root, "skills", provider, "skill.yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the selected runtime provider
	if err != nil {
		return fmt.Errorf("discovery weapon %q unavailable: %w", provider, err)
	}
	var taxonomy skillTaxonomy
	if err := yaml.Unmarshal(raw, &taxonomy); err != nil {
		return fmt.Errorf("discovery weapon %q manifest invalid: %w", provider, err)
	}
	if taxonomy.canonicalRole() != "ranger" {
		return fmt.Errorf("discovery weapon %q is incompatible with Ranger", provider)
	}
	if taxonomy.WeaponContract.IsZero() {
		return fmt.Errorf("discovery weapon contract missing: invocation evidence and fail-closed behavior are required")
	}
	return validateWeaponBoundary(string(domain.SlotDiscovery), "ranger", taxonomy.WeaponContract)
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

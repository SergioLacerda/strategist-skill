package domain

import (
	"fmt"
	"strings"
)

// ValidateRoleWeaponIdentity reconciles the family-aware Role and Weapon
// identities carried by one binding. A compiled registry, when available,
// is authoritative for Role existence and slot ownership; an empty registry
// preserves the legacy Custom path where the external catalog is unavailable.
// Role and Weapon IDs are never compared as if they belonged to one namespace.
func ValidateRoleWeaponIdentity(registry CompiledRegistry, role, slot string, binding SlotBinding) error {
	if err := validateRoleWeaponCoordinates(role, slot, binding); err != nil {
		return err
	}
	weaponIdentity, complete := bindingWeaponIdentity(binding)
	if !complete {
		return nil
	}
	if err := weaponIdentity.Validate(); err != nil {
		return fmt.Errorf("role/weapon identity: %w", err)
	}
	return validateCompiledRoleIdentity(registry, role, slot)
}

func validateRoleWeaponCoordinates(role, slot string, binding SlotBinding) error {
	roleIdentity := CanonicalIdentity{Family: TaxonomyRole, ID: strings.TrimSpace(role)}
	if err := roleIdentity.Validate(); err != nil {
		return fmt.Errorf("role/weapon identity: %w", err)
	}
	if strings.TrimSpace(slot) == "" {
		return fmt.Errorf("role/weapon identity: slot is required")
	}
	if binding.Role != "" && binding.Role != role {
		return fmt.Errorf("role/weapon identity: binding Role %q does not match %q", binding.Role, role)
	}
	if binding.Slot != "" && binding.Slot != slot {
		return fmt.Errorf("role/weapon identity: binding slot %q does not match %q", binding.Slot, slot)
	}
	return nil
}

func bindingWeaponIdentity(binding SlotBinding) (CanonicalIdentity, bool) {
	weaponID, parsedVersion := ParseWeaponRef(strings.TrimSpace(binding.InstalledInstanceID))
	weaponVersion := strings.TrimSpace(binding.WeaponVersion)
	if weaponVersion == "" {
		weaponVersion = parsedVersion
	}
	// Incomplete bindings keep their specialized fail-closed diagnostics in
	// ValidateCustomBinding or the Ranked resolver.
	if strings.TrimSpace(weaponID) == "" || weaponVersion == "" {
		return CanonicalIdentity{}, false
	}
	return CanonicalIdentity{Family: TaxonomyWeapon, ID: weaponID, Version: weaponVersion}, true
}

func validateCompiledRoleIdentity(registry CompiledRegistry, role, slot string) error {
	if len(registry.Roles) == 0 {
		return nil
	}
	compiledRole, ok := registry.Role(role)
	if !ok {
		return fmt.Errorf("role/weapon identity: Role %q is not present in the compiled registry", role)
	}
	if compiledRole.Slot != slot {
		return fmt.Errorf("role/weapon identity: Role %q owns slot %q, not %q", role, compiledRole.Slot, slot)
	}
	return nil
}

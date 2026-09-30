package domain

import (
	"fmt"
	"strings"
)

func validateCompiledWeapons(weapons []CompiledWeapon) error {
	seen := make(map[string]struct{}, len(weapons))
	for _, weapon := range weapons {
		if err := validateCompiledWeapon(weapon); err != nil {
			return err
		}
		if _, duplicate := seen[weapon.Identity()]; duplicate {
			return fmt.Errorf("compiled registry: duplicate Weapon %q", weapon.Identity())
		}
		seen[weapon.Identity()] = struct{}{}
	}
	return nil
}

func validateCompiledWeapon(weapon CompiledWeapon) error {
	if strings.TrimSpace(weapon.ID) == "" {
		return fmt.Errorf("compiled registry: Weapon id is required")
	}
	if strings.TrimSpace(weapon.Version) == "" {
		return fmt.Errorf("compiled registry: Weapon %q version is required", weapon.ID)
	}
	if strings.TrimSpace(weapon.Digest) == "" {
		return fmt.Errorf("compiled registry: Weapon %q digest is required", weapon.Identity())
	}
	if err := weapon.Origin.Validate(); err != nil {
		return fmt.Errorf("compiled registry: Weapon %q: %w", weapon.ID, err)
	}
	if err := weapon.Runtime.Validate(); err != nil {
		return fmt.Errorf("compiled registry: Weapon %q runtime: %w", weapon.ID, err)
	}
	if err := validateCompiledWeaponExecution(weapon); err != nil {
		return err
	}
	return nil
}

func validateCompiledWeaponExecution(weapon CompiledWeapon) error {
	if weapon.Runtime.Kind == RankedRuntimeEmbedded && strings.TrimSpace(weapon.Runtime.ExecutionMode) == "" {
		return fmt.Errorf("compiled registry: Weapon %q execution mode is required for embedded runtime", weapon.ID)
	}
	if weapon.Runtime.ExecutionMode == WeaponExecutionModePromptBridge && strings.TrimSpace(weapon.SourceDigest) == "" {
		return fmt.Errorf("compiled registry: Weapon %q source digest is required for prompt bridge execution", weapon.ID)
	}
	return nil
}

func validateCompiledRoles(roles []CompiledRole) error {
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if _, duplicate := seen[role.ID]; duplicate {
			return fmt.Errorf("compiled registry: duplicate Role %q", role.ID)
		}
		seen[role.ID] = struct{}{}
		if err := validateCompiledRole(role); err != nil {
			return err
		}
	}
	return nil
}

func validateCompiledCompatibility(weapons []CompiledWeapon, roles []CompiledRole, entries []CompiledCompatibility) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if err := validateCompiledCompatibilityEntry(weapons, roles, seen, entry); err != nil {
			return err
		}
	}
	return nil
}

func validateCompiledCompatibilityEntry(weapons []CompiledWeapon, roles []CompiledRole, seen map[string]struct{}, entry CompiledCompatibility) error {
	key := entry.Role + "\x00" + entry.Slot + "\x00" + entry.WeaponID + "\x00" + entry.WeaponVersion
	if _, duplicate := seen[key]; duplicate {
		return fmt.Errorf("compiled registry: duplicate compatibility %s/%s/%s", entry.Role, entry.Slot, entry.WeaponID)
	}
	seen[key] = struct{}{}
	if err := validateCompiledCompatibilityRole(roles, entry); err != nil {
		return err
	}
	weapon, ok := findCompiledWeapon(weapons, entry.WeaponID, entry.WeaponVersion)
	if !ok {
		return fmt.Errorf("compiled registry: compatibility references unknown Weapon %q", WeaponIdentity(entry.WeaponID, entry.WeaponVersion))
	}
	if entry.WeaponDigest != weapon.Digest {
		return fmt.Errorf("compiled registry: compatibility %s/%s Weapon digest mismatch", entry.Role, entry.Slot)
	}
	if entry.Source == "" {
		return fmt.Errorf("compiled registry: compatibility %s/%s source is required", entry.Role, entry.Slot)
	}
	return nil
}

func validateCompiledCompatibilityRole(roles []CompiledRole, entry CompiledCompatibility) error {
	role, ok := findCompiledRole(roles, entry.Role)
	if !ok || role.Slot != entry.Slot {
		return fmt.Errorf("compiled registry: compatibility references unknown Role/slot %s/%s", entry.Role, entry.Slot)
	}
	return nil
}

func validateCompiledRole(role CompiledRole) error {
	if strings.TrimSpace(role.ID) == "" {
		return fmt.Errorf("compiled registry: Role id is required")
	}
	if !IsValidSlot(role.Slot) {
		return fmt.Errorf("compiled registry: Role %q has invalid slot %q", role.ID, role.Slot)
	}
	if strings.TrimSpace(role.ContractDigest) == "" {
		return fmt.Errorf("compiled registry: Role %q contract digest is required", role.ID)
	}
	return nil
}

func findCompiledWeapon(weapons []CompiledWeapon, id, version string) (CompiledWeapon, bool) {
	if strings.TrimSpace(version) == "" {
		return CompiledWeapon{}, false
	}
	for _, weapon := range weapons {
		if weapon.ID == id && weapon.Version == version {
			return weapon, true
		}
	}
	return CompiledWeapon{}, false
}

func findCompiledRole(roles []CompiledRole, id string) (CompiledRole, bool) {
	for _, role := range roles {
		if role.ID == id {
			return role, true
		}
	}
	return CompiledRole{}, false
}

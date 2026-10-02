package domain

import (
	"fmt"
	"strings"
)

func validateCompiledRankedBindings(weapons []CompiledWeapon, roles []CompiledRole, bindings []CompiledRankedBinding) error {
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		key := binding.Role + "\x00" + binding.Slot + "\x00" + WeaponIdentity(binding.WeaponID, binding.WeaponVersion)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("compiled registry: duplicate Ranked binding %s/%s/%s", binding.Role, binding.Slot, WeaponIdentity(binding.WeaponID, binding.WeaponVersion))
		}
		seen[key] = struct{}{}
		if err := validateCompiledRankedBinding(weapons, roles, binding); err != nil {
			return err
		}
	}
	return nil
}

func validateCompiledRankedBinding(weapons []CompiledWeapon, roles []CompiledRole, binding CompiledRankedBinding) error {
	role, weapon, err := rankedBindingReferences(weapons, roles, binding)
	if err != nil {
		return err
	}
	if err := validateRankedBindingDigests(role, weapon, binding); err != nil {
		return err
	}
	if err := validateRankedBindingIdentity(binding); err != nil {
		return err
	}
	return validateRankedBindingRuntime(binding)
}

func rankedBindingReferences(weapons []CompiledWeapon, roles []CompiledRole, binding CompiledRankedBinding) (CompiledRole, CompiledWeapon, error) {
	role, ok := findCompiledRole(roles, binding.Role)
	if !ok || role.Slot != binding.Slot {
		return CompiledRole{}, CompiledWeapon{}, fmt.Errorf("compiled registry: Ranked binding references unknown Role/slot %s/%s", binding.Role, binding.Slot)
	}
	if strings.TrimSpace(binding.WeaponVersion) == "" {
		return CompiledRole{}, CompiledWeapon{}, fmt.Errorf("compiled registry: Ranked binding %s/%s Weapon version is required", binding.Role, binding.Slot)
	}
	weapon, ok := findCompiledWeapon(weapons, binding.WeaponID, binding.WeaponVersion)
	if !ok {
		return CompiledRole{}, CompiledWeapon{}, fmt.Errorf("compiled registry: Ranked binding references unknown Weapon %q", WeaponIdentity(binding.WeaponID, binding.WeaponVersion))
	}
	return role, weapon, nil
}

func validateRankedBindingDigests(role CompiledRole, weapon CompiledWeapon, binding CompiledRankedBinding) error {
	if strings.TrimSpace(binding.WeaponDigest) == "" || strings.TrimSpace(binding.RoleDigest) == "" || strings.TrimSpace(binding.BindingDigest) == "" || strings.TrimSpace(binding.CertificationDigest) == "" {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s has incomplete digests", binding.Role, binding.Slot)
	}
	if binding.WeaponDigest != weapon.Digest || binding.RoleDigest != role.ContractDigest {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s digest mismatch", binding.Role, binding.Slot)
	}
	if binding.ExecutionMode != weapon.Runtime.ExecutionMode || binding.SourceDigest != weapon.SourceDigest {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s execution identity mismatch", binding.Role, binding.Slot)
	}
	return nil
}

func validateRankedBindingIdentity(binding CompiledRankedBinding) error {
	if strings.TrimSpace(binding.ConnectorID) == "" || strings.TrimSpace(binding.Entrypoint) == "" || binding.Generation <= 0 || strings.TrimSpace(binding.Status) == "" {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s has incomplete runtime identity", binding.Role, binding.Slot)
	}
	return nil
}

func validateRankedBindingRuntime(binding CompiledRankedBinding) error {
	if err := binding.Runtime.ValidateActive(); err != nil {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s runtime: %w", binding.Role, binding.Slot, err)
	}
	if binding.Runtime.Kind == RankedRuntimeEmbedded && binding.Runtime.HostAPI != "" {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s Embedded runtime cannot declare HostAPI", binding.Role, binding.Slot)
	}
	if binding.Runtime.Kind == RankedRuntimeHost || binding.Runtime.Kind == RankedRuntimeExecutable {
		return fmt.Errorf("compiled registry: Ranked binding %s/%s cannot use external runtime %q", binding.Role, binding.Slot, binding.Runtime.Kind)
	}
	return nil
}

package main

import (
	"fmt"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
)

func resolveMissionInvocationWeapon(input missionadapter.InvocationBuildInput) (domain.RoleWeaponBinding, domain.CompiledWeapon, []byte, string, error) {
	active, lock, registry, err := loadMissionInvocationState(input.Root)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", err
	}
	binding, weapon, err := resolveEmbeddedInvocationBinding(active, lock, registry, input)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", err
	}
	payload, sourceDigest, err := readMissionInvocationPayload(binding)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", fmt.Errorf("embedded_payload_mismatch: %w", err)
	}
	if sourceDigest != binding.SourceDigest || sourceDigest != weapon.SourceDigest {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", fmt.Errorf("embedded_payload_mismatch: payload digest does not match compiled binding")
	}
	return binding, weapon, payload, sourceDigest, nil
}

func readMissionInvocationPayload(binding domain.RoleWeaponBinding) ([]byte, string, error) {
	extractor := strategistembed.Extractor{}
	if binding.ConnectorID == "strategist-native-role" {
		payload, digest, err := extractor.ReadEmbeddedNativeRolePayload(binding.WeaponID)
		if err != nil {
			return nil, "", fmt.Errorf("read native Role payload: %w", err)
		}
		return payload, digest, nil
	}
	payload, digest, err := extractor.ReadEmbeddedWeaponPayload(binding.WeaponID, binding.WeaponVersion)
	if err != nil {
		return nil, "", fmt.Errorf("read embedded Weapon payload: %w", err)
	}
	return payload, digest, nil
}

func resolveEmbeddedInvocationBinding(active domain.ActiveConfig, lock domain.PluginLockFile, registry domain.CompiledRegistry, input missionadapter.InvocationBuildInput) (domain.RoleWeaponBinding, domain.CompiledWeapon, error) {
	binding, err := domain.ResolveRoleWeaponBinding(active, lock, registry, input.Role, input.Slot)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, fmt.Errorf("role_invocation_failed: %w", err)
	}
	if err := validateEmbeddedInvocationBinding(binding); err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, err
	}
	weapon, ok := registry.Weapon(binding.WeaponID, binding.WeaponVersion)
	if !ok {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, fmt.Errorf("compiled_binding_missing: Weapon %q is not in the compiled registry", domain.WeaponIdentity(binding.WeaponID, binding.WeaponVersion))
	}
	return binding, weapon, nil
}

func validateEmbeddedInvocationBinding(binding domain.RoleWeaponBinding) error {
	if binding.Mode != domain.SlotBindingModeRanked {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports Ranked Embedded prompt-bridge bindings")
	}
	if binding.RuntimeKind == domain.RankedRuntimeOpenSpecRoot {
		return fmt.Errorf("role_invocation_failed: Ranked openspec_root bindings execute through their declared private runtime and mission normalize-openspec, not mission invoke")
	}
	if binding.RuntimeKind != domain.RankedRuntimeEmbedded {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports runtime kind %q, got %q", domain.RankedRuntimeEmbedded, binding.RuntimeKind)
	}
	return nil
}

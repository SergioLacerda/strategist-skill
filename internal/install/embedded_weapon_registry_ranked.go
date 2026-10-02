package install

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func buildCompiledRankedBindings(catalog pluginCatalog, weapons []domain.CompiledWeapon, roles []domain.CompiledRole) ([]domain.CompiledRankedBinding, error) {
	bindings := make([]domain.CompiledRankedBinding, 0)
	for _, provider := range catalog.Providers {
		if !provider.Ranked || provider.CertificationDigest == "" || provider.CompatibilitySource == "external" {
			continue
		}
		providerBindings, err := compileProviderRankedBindings(provider, weapons, roles)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, providerBindings...)
	}
	sort.Slice(bindings, func(i, j int) bool {
		left := bindings[i].Role + "\x00" + bindings[i].Slot + "\x00" + domain.WeaponIdentity(bindings[i].WeaponID, bindings[i].WeaponVersion)
		right := bindings[j].Role + "\x00" + bindings[j].Slot + "\x00" + domain.WeaponIdentity(bindings[j].WeaponID, bindings[j].WeaponVersion)
		return left < right
	})
	return bindings, nil
}

func compileProviderRankedBindings(provider pluginCatalogProvider, weapons []domain.CompiledWeapon, roles []domain.CompiledRole) ([]domain.CompiledRankedBinding, error) {
	weapon, ok := compiledWeaponByIdentity(weapons, provider.ID, providerVersionOrDefault(provider.Version))
	if !ok {
		return nil, fmt.Errorf("ranked provider %q is absent from compiled Weapons", domain.WeaponIdentity(provider.ID, providerVersionOrDefault(provider.Version)))
	}
	bindings := make([]domain.CompiledRankedBinding, 0, len(providerRoles(provider)))
	for _, roleID := range providerRoles(provider) {
		role, ok := compiledRoleByID(roles, roleID)
		if !ok {
			return nil, fmt.Errorf("ranked provider %q references unknown Role %q", provider.ID, roleID)
		}
		binding, err := compileRankedBinding(provider, weapon, role)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func compileRankedBinding(provider pluginCatalogProvider, weapon domain.CompiledWeapon, role domain.CompiledRole) (domain.CompiledRankedBinding, error) {
	bindingBytes, err := yaml.Marshal(struct {
		Role, Slot, WeaponID, WeaponVersion, WeaponDigest, SourceDigest, RoleDigest, ExecutionMode, CertificationDigest string
	}{role.ID, role.Slot, weapon.ID, weapon.Version, weapon.Digest, weapon.SourceDigest, weapon.Runtime.ExecutionMode, role.ContractDigest, provider.CertificationDigest})
	if err != nil {
		return domain.CompiledRankedBinding{}, fmt.Errorf("marshal Ranked binding %s/%s: %w", role.ID, role.Slot, err)
	}
	sum := sha256.Sum256(bindingBytes)
	return domain.CompiledRankedBinding{
		Role: role.ID, Slot: role.Slot, WeaponID: weapon.ID, WeaponVersion: weapon.Version, WeaponDigest: weapon.Digest, RoleDigest: role.ContractDigest,
		SourceDigest: weapon.SourceDigest, ExecutionMode: weapon.Runtime.ExecutionMode,
		BindingDigest: fmt.Sprintf("sha256:%x", sum), CertificationDigest: provider.CertificationDigest,
		ConnectorID: weapon.ConnectorID, Runtime: weapon.Runtime, Entrypoint: compiledEntrypoint(provider.SupportedSlots, provider.ID),
		Generation: provider.RankedBindingGeneration, Status: provider.RankedBindingStatus,
	}, nil
}

func compiledEntrypoint(slots []string, providerID string) string {
	for _, slot := range slots {
		switch slot {
		case string(domain.SlotDiscovery):
			return "discover"
		case string(domain.SlotRefinement):
			return "refine"
		case string(domain.SlotExecution):
			return "execute"
		}
	}
	if providerID == "sniper" {
		return "execute"
	}
	return "invoke"
}

func compiledWeaponByIdentity(weapons []domain.CompiledWeapon, id, version string) (domain.CompiledWeapon, bool) {
	for _, weapon := range weapons {
		if weapon.ID == id && weapon.Version == version {
			return weapon, true
		}
	}
	return domain.CompiledWeapon{}, false
}

func compiledRoleByID(roles []domain.CompiledRole, id string) (domain.CompiledRole, bool) {
	for _, role := range roles {
		if role.ID == id {
			return role, true
		}
	}
	return domain.CompiledRole{}, false
}

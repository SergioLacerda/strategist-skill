package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// buildCompiledRegistry derives the immutable Role/Weapon graph from the
// already-ingested catalog and canonical native Role files. It intentionally
// does not create a second Weapon source of truth.
func buildCompiledRegistry(catalog pluginCatalog, defaultsRoot string) (domain.CompiledRegistry, error) {
	roles, err := buildCompiledRoles(defaultsRoot)
	if err != nil {
		return domain.CompiledRegistry{}, err
	}
	weapons, err := buildCompiledWeapons(catalog, defaultsRoot)
	if err != nil {
		return domain.CompiledRegistry{}, err
	}
	bindings, err := buildCompiledRankedBindings(catalog, weapons, roles)
	if err != nil {
		return domain.CompiledRegistry{}, err
	}
	registry := domain.CompiledRegistry{
		SchemaVersion:   domain.CompiledRegistrySchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Weapons:         weapons,
		Roles:           roles,
		Compatibility:   buildCompiledCompatibility(weapons, roles),
		RankedBindings:  bindings,
	}
	if err := registry.Validate(); err != nil {
		return domain.CompiledRegistry{}, fmt.Errorf("validate compiled registry: %w", err)
	}
	return registry, nil
}

func buildCompiledRoles(defaultsRoot string) ([]domain.CompiledRole, error) {
	roleMapRaw, err := os.ReadFile(filepath.Join(defaultsRoot, roleSlotMapPath)) //nolint:gosec // build input is caller-configured repository data
	if err != nil {
		return nil, fmt.Errorf("read compiled Role map: %w", err)
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(roleMapRaw, &roleMap); err != nil {
		return nil, fmt.Errorf("parse compiled Role map: %w", err)
	}
	if err := roleMap.Validate(); err != nil {
		return nil, fmt.Errorf("validate compiled Role map: %w", err)
	}

	roles := make([]domain.CompiledRole, 0, len(roleMap))
	for slot, roleID := range roleMap {
		role, err := compileRole(defaultsRoot, slot, roleID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })
	return roles, nil
}

func compileRole(defaultsRoot, slot, roleID string) (domain.CompiledRole, error) {
	rolePath := filepath.Join(defaultsRoot, "roles", roleID+".yaml")
	raw, err := os.ReadFile(rolePath) //nolint:gosec // build input is below caller-configured defaults root
	if err != nil {
		return domain.CompiledRole{}, fmt.Errorf("read compiled Role %q: %w", roleID, err)
	}
	var cfg domain.RoleConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return domain.CompiledRole{}, fmt.Errorf("parse compiled Role %q: %w", roleID, err)
	}
	if err := cfg.Validate(); err != nil {
		return domain.CompiledRole{}, fmt.Errorf("validate compiled Role %q: %w", roleID, err)
	}
	contract := domain.RoleContractFromConfig(cfg, domain.DefaultRoleRegistry().HandoffSchemaOf(roleID))
	contractBytes, err := yaml.Marshal(contract)
	if err != nil {
		return domain.CompiledRole{}, fmt.Errorf("marshal compiled Role %q: %w", roleID, err)
	}
	sum := sha256.Sum256(contractBytes)
	return domain.CompiledRole{ID: roleID, Slot: slot, ContractDigest: fmt.Sprintf("sha256:%x", sum), HandoffSchema: contract.HandoffSchema}, nil
}

func buildCompiledWeapons(catalog pluginCatalog, defaultsRoot string) ([]domain.CompiledWeapon, error) {
	weapons := make([]domain.CompiledWeapon, 0, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		weapon, include, err := compileCatalogWeapon(provider, defaultsRoot)
		if err != nil {
			return nil, err
		}
		if include {
			weapons = append(weapons, weapon)
		}
	}
	sort.Slice(weapons, func(i, j int) bool { return weapons[i].Identity() < weapons[j].Identity() })
	return weapons, nil
}

func compileCatalogWeapon(provider pluginCatalogProvider, defaultsRoot string) (domain.CompiledWeapon, bool, error) {
	if provider.CompatibilitySource == "external" {
		return domain.CompiledWeapon{}, false, nil
	}
	sourceDigest, err := embeddedWeaponSourceDigestAt(catalogWeaponPayloadPath(provider, defaultsRoot))
	if err != nil {
		return domain.CompiledWeapon{}, false, err
	}
	if (provider.CompatibilitySource == "embedded" || provider.CompatibilitySource == "native_role") && sourceDigest == "" {
		return domain.CompiledWeapon{}, false, fmt.Errorf("embedded_payload_missing: Weapon %q has no SKILL.md payload", provider.ID)
	}
	runtime, connectorID := bindingRuntimeIdentity(provider)
	return domain.CompiledWeapon{
		ID: provider.ID, Version: providerVersionOrDefault(provider.Version), Digest: catalogProviderDigest(provider),
		SourceDigest: sourceDigest,
		Origin:       domain.WeaponOriginEmbedded, Runtime: runtime, ConnectorID: connectorID,
		Entrypoint: compiledEntrypoint(provider.SupportedSlots, provider.ID), SupportedRoles: providerRoles(provider), SupportedSlots: append([]string(nil), provider.SupportedSlots...),
	}, true, nil
}

func catalogWeaponPayloadPath(provider pluginCatalogProvider, defaultsRoot string) string {
	if provider.sourcePath != "" {
		return filepath.Join(provider.sourcePath, "SKILL.md")
	}
	if provider.CompatibilitySource == "native_role" {
		return filepath.Join(defaultsRoot, "internal_skills", provider.ID, "SKILL.md")
	}
	return filepath.Join(defaultsRoot, "skills", providerPayloadDirName(provider), "SKILL.md")
}

func buildCompiledCompatibility(weapons []domain.CompiledWeapon, roles []domain.CompiledRole) []domain.CompiledCompatibility {
	entries := make([]domain.CompiledCompatibility, 0)
	for _, role := range roles {
		entries = append(entries, compiledCompatibilityForRole(weapons, role)...)
	}
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i].Role + "\x00" + entries[i].Slot + "\x00" + domain.WeaponIdentity(entries[i].WeaponID, entries[i].WeaponVersion)
		right := entries[j].Role + "\x00" + entries[j].Slot + "\x00" + domain.WeaponIdentity(entries[j].WeaponID, entries[j].WeaponVersion)
		return left < right
	})
	return entries
}

func compiledCompatibilityForRole(weapons []domain.CompiledWeapon, role domain.CompiledRole) []domain.CompiledCompatibility {
	entries := make([]domain.CompiledCompatibility, 0)
	for _, weapon := range weapons {
		if !containsString(weapon.SupportedRoles, role.ID) || !containsString(weapon.SupportedSlots, role.Slot) {
			continue
		}
		entries = append(entries, domain.CompiledCompatibility{
			Role: role.ID, Slot: role.Slot, WeaponID: weapon.ID, WeaponVersion: weapon.Version,
			WeaponDigest: weapon.Digest, HandoffSchema: role.HandoffSchema, Source: "declared",
		})
	}
	return entries
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

package plugincatalog

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

// ProviderContractsForRole projects every catalog entry whose canonical_role
// matches roleName into a ProviderContract, deriving ProviderSource and
// MaterializationState from existing catalog fields instead of adding a
// second provider registry (Decision 4).
func ProviderContractsForRole(catalog PluginCatalog, roleName string) []domain.ProviderContract {
	contracts := make([]domain.ProviderContract, 0)
	for _, provider := range catalog.Providers {
		if !ProviderHasRole(provider, roleName) {
			continue
		}
		contracts = append(contracts, ProviderContractFromCatalogEntry(provider))
	}
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].ID != contracts[j].ID {
			return contracts[i].ID < contracts[j].ID
		}
		return domain.CompareWeaponVersions(contracts[i].Version, contracts[j].Version) < 0
	})
	return contracts
}

// ProviderContractFromCatalogEntry projects a catalog provider onto the provider contract used for role affinity checks.
func ProviderContractFromCatalogEntry(provider Provider) domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 "strategist-provider-contract/v1",
		ID:                            provider.ID,
		Version:                       ProviderVersionOrDefault(provider.Version),
		ProviderSchemaVersion:         providerSchemaVersionOrDefault(provider.SchemaVersion),
		CanonicalRole:                 canonicalRoleOrDefault(provider),
		Roles:                         ProviderRoles(provider),
		RiskScore:                     provider.RiskScore,
		Source:                        providerSourceFromCompatibilitySource(provider.CompatibilitySource),
		Materialization:               materializationFromCatalogEntry(provider),
		Default:                       provider.Default,
		Ranked:                        provider.Ranked,
		CertificationDigest:           provider.CertificationDigest,
		RankedBindingGeneration:       provider.RankedBindingGeneration,
		RankedBindingStatus:           provider.RankedBindingStatus,
		Runtime:                       provider.Runtime,
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
		SupportedHandoffSchemas:       provider.SupportedHandoffSchemas,
	}
}

// ProviderRoles returns the roles a catalog provider declares, normalised.
func ProviderRoles(provider Provider) []string {
	if len(provider.Roles) > 0 {
		return NormalizeRoles(provider.Roles)
	}
	if role := canonicalRoleOrDefault(provider); role != "" {
		return []string{role}
	}
	return nil
}

// ProviderHasRole reports whether provider declares role.
func ProviderHasRole(provider Provider, role string) bool {
	for _, candidateRole := range ProviderRoles(provider) {
		if candidateRole == role {
			return true
		}
	}
	return false
}

// canonicalRoleOrDefault returns the catalog entry's declared canonical_role,
// or falls back to its own ID for a native_role entry with the field left
// unset — a native role's catalog id (e.g. "sniper", "archivist") already
// names the role it embodies (see roles/<id>.yaml), so no separate
// declaration is required for it to participate in role compatibility
// resolution.
func canonicalRoleOrDefault(provider Provider) string {
	if provider.CanonicalRole != "" {
		return provider.CanonicalRole
	}
	if provider.CompatibilitySource == names.CompatibilityNativeRole {
		return provider.ID
	}
	return ""
}

func providerSourceFromCompatibilitySource(compatibilitySource string) domain.ProviderSource {
	switch compatibilitySource {
	case names.CompatibilityEmbedded:
		return domain.ProviderSourceEmbedded
	case names.CompatibilityNativeRole:
		return domain.ProviderSourceNativeRole
	case names.CompatibilityExternal:
		return domain.ProviderSourceExternal
	default:
		// Unrecognized compatibility_source values still resolve to External —
		// conservative default, never silently promoted to Embedded/NativeRole.
		return domain.ProviderSourceExternal
	}
}

func materializationFromCatalogEntry(provider Provider) domain.MaterializationState {
	if provider.CompatibilitySource == names.CompatibilityNativeRole {
		return domain.MaterializationActive
	}
	if provider.Installable {
		return domain.MaterializationInstalled
	}
	return domain.MaterializationUnavailable
}

func providerSchemaVersionOrDefault(v string) string {
	if v == "" {
		return "1"
	}
	return v
}

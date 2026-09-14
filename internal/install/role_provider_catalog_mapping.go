package install

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// providerContractsForRole projects every catalog entry whose canonical_role
// matches roleName into a ProviderContract, deriving ProviderSource and
// MaterializationState from existing catalog fields instead of adding a
// second provider registry (Decision 4).
func providerContractsForRole(catalog pluginCatalog, roleName string) []domain.ProviderContract {
	contracts := make([]domain.ProviderContract, 0)
	for _, provider := range catalog.Providers {
		if canonicalRoleOrDefault(provider) != roleName {
			continue
		}
		contracts = append(contracts, providerContractFromCatalogEntry(provider))
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].ID < contracts[j].ID })
	return contracts
}

func providerContractFromCatalogEntry(provider pluginCatalogProvider) domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 "strategist-provider-contract/v1",
		ID:                            provider.ID,
		Version:                       providerVersionOrDefault(provider.Version),
		ProviderSchemaVersion:         providerSchemaVersionOrDefault(provider.SchemaVersion),
		CanonicalRole:                 canonicalRoleOrDefault(provider),
		RiskScore:                     provider.RiskScore,
		Source:                        providerSourceFromCompatibilitySource(provider.CompatibilitySource),
		Materialization:               materializationFromCatalogEntry(provider),
		Default:                       provider.Default,
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
	}
}

// canonicalRoleOrDefault returns the catalog entry's declared canonical_role,
// or falls back to its own ID for a native_role entry with the field left
// unset — a native role's catalog id (e.g. "sniper", "archivist") already
// names the role it embodies (see roles/<id>.yaml), so no separate
// declaration is required for it to participate in role compatibility
// resolution.
func canonicalRoleOrDefault(provider pluginCatalogProvider) string {
	if provider.CanonicalRole != "" {
		return provider.CanonicalRole
	}
	if provider.CompatibilitySource == "native_role" {
		return provider.ID
	}
	return ""
}

func providerSourceFromCompatibilitySource(compatibilitySource string) domain.ProviderSource {
	switch compatibilitySource {
	case "embedded":
		return domain.ProviderSourceEmbedded
	case "native_role":
		return domain.ProviderSourceNativeRole
	case "external":
		return domain.ProviderSourceExternal
	default:
		// Unrecognized compatibility_source values still resolve to External —
		// conservative default, never silently promoted to Embedded/NativeRole.
		return domain.ProviderSourceExternal
	}
}

func materializationFromCatalogEntry(provider pluginCatalogProvider) domain.MaterializationState {
	if provider.CompatibilitySource == "native_role" {
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

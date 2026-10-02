package install

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// providerIdentity is the catalog identity of one provider, "id@version"
// (ADR-0061 Decision 8). An entry that omits its version keeps the historic
// default so existing catalogs stay valid.
func providerIdentity(provider pluginCatalogProvider) string {
	return domain.WeaponIdentity(provider.ID, providerVersionOrDefault(provider.Version))
}

// skillIdentity is the identity an ingested package will carry in the catalog.
func skillIdentity(skill IngestedSkill) string {
	return domain.WeaponIdentity(skill.ID, providerVersionOrDefault(skill.Package.Version))
}

// findCatalogProvider resolves a provider by id alone only when exactly one
// version of it exists. With several versions an id never resolves: there is no
// first-match and no "latest" (ADR-0060 Decision 7).
func findCatalogProvider(catalog pluginCatalog, providerID string) (pluginCatalogProvider, bool) {
	var found pluginCatalogProvider
	count := 0
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			found = provider
			count++
		}
	}
	return found, count == 1
}

// findCatalogProviderVersion resolves one exact id@version. An empty version
// never resolves.
func findCatalogProviderVersion(catalog pluginCatalog, providerID, version string) (pluginCatalogProvider, bool) {
	if version == "" {
		return pluginCatalogProvider{}, false
	}
	for _, provider := range catalog.Providers {
		if provider.ID == providerID && providerVersionOrDefault(provider.Version) == version {
			return provider, true
		}
	}
	return pluginCatalogProvider{}, false
}

// catalogProviderVersions lists the versions catalogued for one id, sorted.
func catalogProviderVersions(catalog pluginCatalog, providerID string) []string {
	var versions []string
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			versions = append(versions, providerVersionOrDefault(provider.Version))
		}
	}
	sort.Slice(versions, func(i, j int) bool { return domain.CompareWeaponVersions(versions[i], versions[j]) < 0 })
	return versions
}

// validateCatalogIdentities rejects the same id@version catalogued twice.
func validateCatalogIdentities(providers []pluginCatalogProvider) error {
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		identity := providerIdentity(provider)
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf("plugin catalog: duplicate provider %q", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

// skillPayloadDirName is the skills/ directory of one ingested package version.
func skillPayloadDirName(skill IngestedSkill) string {
	return domain.WeaponPayloadDirName(skill.ID, skill.Package.Version)
}

// providerPayloadDirName is the skills/ directory of one catalogued provider.
func providerPayloadDirName(provider pluginCatalogProvider) string {
	return domain.WeaponPayloadDirName(provider.ID, provider.Version)
}

func sortCatalogProviders(providers []pluginCatalogProvider) {
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].ID != providers[j].ID {
			return providers[i].ID < providers[j].ID
		}
		return domain.CompareWeaponVersions(providerVersionOrDefault(providers[i].Version), providerVersionOrDefault(providers[j].Version)) < 0
	})
}

// findCatalogProviderRef resolves an active.yaml slot value: the whole string as
// an id first (a Custom package id such as "team-skill@1.2.0" carries its own
// version), then "id@version" exactly. A plain id resolves only while one
// version exists, never as a first match or "latest" (ADR-0061 Decision 9).
func findCatalogProviderRef(catalog pluginCatalog, ref string) (pluginCatalogProvider, bool) {
	if provider, ok := findCatalogProvider(catalog, ref); ok {
		return provider, true
	}
	id, version := domain.ParseWeaponRef(ref)
	if version == "" {
		return pluginCatalogProvider{}, false
	}
	return findCatalogProviderVersion(catalog, id, version)
}

// unresolvedRefHint names the catalogued versions of an id whose reference did
// not resolve, so the operator sees which id@version to select.
func unresolvedRefHint(catalog pluginCatalog, ref string) string {
	id, _ := domain.ParseWeaponRef(ref)
	versions := catalogProviderVersions(catalog, id)
	if len(versions) < 2 {
		return ""
	}
	refs := make([]string, 0, len(versions))
	for _, version := range versions {
		refs = append(refs, domain.WeaponIdentity(id, version))
	}
	return fmt.Sprintf(" (several versions are catalogued: %s; select one as id@version)", strings.Join(refs, ", "))
}

// catalogKnowsRef reports whether the catalog lists ref at all, including a
// plain id that is ambiguous across versions: such an id is a catalog Weapon,
// never a Custom package to project.
func catalogKnowsRef(catalog pluginCatalog, ref string) bool {
	id, version := domain.ParseWeaponRef(ref)
	for _, provider := range catalog.Providers {
		if provider.ID == ref {
			return true
		}
		if provider.ID == id && (version == "" || providerVersionOrDefault(provider.Version) == version) {
			return true
		}
	}
	return false
}

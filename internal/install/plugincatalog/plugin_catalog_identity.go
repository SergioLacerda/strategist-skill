package plugincatalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ProviderIdentity is the catalog identity of one provider, "id@version"
// (ADR-0061 Decision 8). An entry that omits its version keeps the historic
// default so existing catalogs stay valid.
func ProviderIdentity(provider Provider) string {
	return domain.WeaponIdentity(provider.ID, ProviderVersionOrDefault(provider.Version))
}

// SkillIdentity is the identity an ingested package will carry in the catalog.
func SkillIdentity(skill IngestedSkill) string {
	return domain.WeaponIdentity(skill.ID, ProviderVersionOrDefault(skill.Package.Version))
}

// FindCatalogProvider resolves a provider by id alone only when exactly one
// version of it exists. With several versions an id never resolves: there is no
// first-match and no "latest" (ADR-0060 Decision 7).
func FindCatalogProvider(catalog PluginCatalog, providerID string) (Provider, bool) {
	var found Provider
	count := 0
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			found = provider
			count++
		}
	}
	return found, count == 1
}

// FindCatalogProviderVersion resolves one exact id@version. An empty version
// never resolves.
func FindCatalogProviderVersion(catalog PluginCatalog, providerID, version string) (Provider, bool) {
	if version == "" {
		return Provider{}, false
	}
	for _, provider := range catalog.Providers {
		if provider.ID == providerID && ProviderVersionOrDefault(provider.Version) == version {
			return provider, true
		}
	}
	return Provider{}, false
}

// CatalogProviderVersions lists the versions catalogued for one id, sorted.
func CatalogProviderVersions(catalog PluginCatalog, providerID string) []string {
	var versions []string
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			versions = append(versions, ProviderVersionOrDefault(provider.Version))
		}
	}
	sort.Slice(versions, func(i, j int) bool { return domain.CompareWeaponVersions(versions[i], versions[j]) < 0 })
	return versions
}

// validateCatalogIdentities rejects the same id@version catalogued twice.
func validateCatalogIdentities(providers []Provider) error {
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		identity := ProviderIdentity(provider)
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf("plugin catalog: duplicate provider %q", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

// SkillPayloadDirName is the skills/ directory of one ingested package version.
func SkillPayloadDirName(skill IngestedSkill) string {
	return domain.WeaponPayloadDirName(skill.ID, skill.Package.Version)
}

// ProviderPayloadDirName is the skills/ directory of one catalogued provider.
func ProviderPayloadDirName(provider Provider) string {
	return domain.WeaponPayloadDirName(provider.ID, provider.Version)
}

// SortCatalogProviders orders providers by id and then by version, in place.
func SortCatalogProviders(providers []Provider) {
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].ID != providers[j].ID {
			return providers[i].ID < providers[j].ID
		}
		return domain.CompareWeaponVersions(ProviderVersionOrDefault(providers[i].Version), ProviderVersionOrDefault(providers[j].Version)) < 0
	})
}

// FindCatalogProviderRef resolves an active.yaml slot value: the whole string as
// an id first (a Custom package id such as "team-skill@1.2.0" carries its own
// version), then "id@version" exactly. A plain id resolves only while one
// version exists, never as a first match or "latest" (ADR-0061 Decision 9).
func FindCatalogProviderRef(catalog PluginCatalog, ref string) (Provider, bool) {
	if provider, ok := FindCatalogProvider(catalog, ref); ok {
		return provider, true
	}
	id, version := domain.ParseWeaponRef(ref)
	if version == "" {
		return Provider{}, false
	}
	return FindCatalogProviderVersion(catalog, id, version)
}

// UnresolvedRefHint names the catalogued versions of an id whose reference did
// not resolve, so the operator sees which id@version to select.
func UnresolvedRefHint(catalog PluginCatalog, ref string) string {
	id, _ := domain.ParseWeaponRef(ref)
	versions := CatalogProviderVersions(catalog, id)
	if len(versions) < 2 {
		return ""
	}
	refs := make([]string, 0, len(versions))
	for _, version := range versions {
		refs = append(refs, domain.WeaponIdentity(id, version))
	}
	return fmt.Sprintf(" (several versions are catalogued: %s; select one as id@version)", strings.Join(refs, ", "))
}

// CatalogKnowsRef reports whether the catalog lists ref at all, including a
// plain id that is ambiguous across versions: such an id is a catalog Weapon,
// never a Custom package to project.
func CatalogKnowsRef(catalog PluginCatalog, ref string) bool {
	id, version := domain.ParseWeaponRef(ref)
	for _, provider := range catalog.Providers {
		if provider.ID == ref {
			return true
		}
		if provider.ID == id && (version == "" || ProviderVersionOrDefault(provider.Version) == version) {
			return true
		}
	}
	return false
}

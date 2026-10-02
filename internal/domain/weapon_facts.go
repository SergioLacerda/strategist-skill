package domain

import (
	"errors"
)

// Where a resolved Weapon manifest came from.
const (
	// WeaponFactsSourceCatalog is the authority: the materialized
	// plugins/catalog.yaml entry.
	WeaponFactsSourceCatalog = "catalog"
	// WeaponFactsSourceAdapter is the adapter.yaml of a package added with
	// `provider add` and bound with mode custom.
	WeaponFactsSourceAdapter = "adapter"
	// WeaponFactsSourceCompatView is the generated skills/<id>/skill.yaml
	// compatibility view, consulted only for a provider the catalog does not
	// list (ADR-0030: a migration label, never an authority).
	WeaponFactsSourceCompatView = "compat_view"
)

// ErrWeaponFactsNotFound means neither the catalog nor the compatibility view
// describes the provider.
var ErrWeaponFactsNotFound = errors.New("weapon facts not found")

// ErrWeaponFactsAmbiguous means a plain id names several catalogued versions;
// the reference must be written id@version (ADR-0061 Decision 9).
var ErrWeaponFactsAmbiguous = errors.New("weapon reference is ambiguous across versions")

// WeaponFacts is the risk, role and runtime facts preflight and role
// validation need about one Weapon, independent of where they were read from.
type WeaponFacts struct {
	ID string
	// Version is the catalogued version; empty for a provider known only through
	// the compat view. Together with ID it names the skills/<id>@<version>/ payload.
	Version        string
	Source         string
	RiskScore      string
	CanonicalRole  string
	Roles          []string
	ScratchRoot    string
	WeaponContract WeaponContract
	// CompatibilitySource is the catalog classification (native_role, embedded,
	// external); empty for a provider known only through the compat view.
	CompatibilitySource string
	Installable         bool
	SupportedSlots      []string
	RuntimeKind         string
	RuntimeRoot         string
	// RuntimeHostAPI is the host API the catalog runtime block declares, empty
	// when it declares none.
	RuntimeHostAPI string
	// RequestedPermissions is what the Weapon asks to be granted; only an
	// adapter.yaml declares it today (the catalog entry declares none).
	RequestedPermissions []PluginPermission
	// Entrypoints are the entrypoints an adapter.yaml declares; the catalog and the
	// compat view declare none here.
	Entrypoints []string
}

// weaponFactsDoc is the subset of a catalog entry, or of a generated
// skills/<id>/skill.yaml, that WeaponFacts reads. Both carry these keys; the
// nested specialization_taxonomy shape is accepted for compiled runtime copies.
type weaponFactsDoc struct {
	ID                   string             `yaml:"id"`
	Version              string             `yaml:"version"`
	RiskScore            string             `yaml:"risk_score"`
	CanonicalRole        string             `yaml:"canonical_role"`
	Roles                []string           `yaml:"roles"`
	ScratchRoot          string             `yaml:"scratch_root"`
	WeaponContract       WeaponContract     `yaml:"weapon_contract"`
	CompatibilitySource  string             `yaml:"compatibility_source"`
	Installable          bool               `yaml:"installable"`
	SupportedSlots       []string           `yaml:"supported_slots"`
	RequestedPermissions []PluginPermission `yaml:"requested_permissions"`
	Runtime              struct {
		Kind    string `yaml:"kind"`
		Root    string `yaml:"root"`
		HostAPI string `yaml:"host_api"`
	} `yaml:"runtime"`
	SpecializationTaxonomy struct {
		CanonicalRole string `yaml:"canonical_role"`
	} `yaml:"specialization_taxonomy"`
}

func (d weaponFactsDoc) manifest(source string) WeaponFacts {
	role := d.CanonicalRole
	if role == "" {
		role = d.SpecializationTaxonomy.CanonicalRole
	}
	roles := d.Roles
	if len(roles) == 0 && role != "" {
		roles = []string{role}
	}
	return WeaponFacts{
		ID: d.ID, Version: d.Version, Source: source, RiskScore: d.RiskScore, CanonicalRole: role,
		Roles: roles, ScratchRoot: d.ScratchRoot, WeaponContract: d.WeaponContract,
		CompatibilitySource: d.CompatibilitySource, Installable: d.Installable,
		SupportedSlots: d.SupportedSlots, RuntimeKind: d.Runtime.Kind, RuntimeRoot: d.Runtime.Root, RuntimeHostAPI: d.Runtime.HostAPI,
		RequestedPermissions: d.RequestedPermissions,
	}
}

// ResolveWeaponFacts returns the manifest of provider under a .strategist
// root. The catalog entry is the authority; the adapter.yaml of a package bound
// with mode custom comes next; the generated compatibility view is read last,
// and only for a provider neither of them describes. An unreadable catalog is an
// error: a broken authority never silently falls back.
func ResolveWeaponFacts(strategistRoot, provider string) (WeaponFacts, error) {
	return ResolveWeaponFactsFrom(strategistRoot, provider, nil)
}

// ResolveWeaponFactsFrom is ResolveWeaponFacts for a caller that has already
// read the provider's compatibility view: when the catalog does not list the
// provider, compat is parsed instead of being read from disk again. A nil compat
// reads skills/<id>/skill.yaml.
func ResolveWeaponFactsFrom(strategistRoot, provider string, compat []byte) (WeaponFacts, error) {
	facts, found, err := factsFromCatalog(strategistRoot, provider)
	if err != nil || found {
		return facts, err
	}
	if facts, found, err = factsFromAdapter(strategistRoot, provider); err != nil || found {
		return facts, err
	}
	if compat != nil {
		return parseCompatView(provider, compat)
	}
	return factsFromCompatView(strategistRoot, provider)
}

// ResolveCatalogWeaponFacts returns the catalog entry of provider, if the
// catalog lists it. Unlike ResolveWeaponFacts it never falls back to an
// adapter.yaml or to the compat view; slot resolution uses it to tell a
// cataloged Weapon from a provider the catalog does not know.
func ResolveCatalogWeaponFacts(strategistRoot, provider string) (WeaponFacts, bool, error) {
	return factsFromCatalog(strategistRoot, provider)
}

// ListCatalogWeaponFacts returns every entry of the plugin catalog, in catalog
// order. An absent catalog yields none.
func ListCatalogWeaponFacts(strategistRoot string) ([]WeaponFacts, error) {
	docs, err := readCatalogDocs(strategistRoot)
	if err != nil {
		return nil, err
	}
	out := make([]WeaponFacts, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.manifest(WeaponFactsSourceCatalog))
	}
	return out, nil
}

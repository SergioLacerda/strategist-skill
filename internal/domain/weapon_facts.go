package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Where a resolved Weapon manifest came from.
const (
	// WeaponFactsSourceCatalog is the authority: the materialized
	// plugins/catalog.yaml entry.
	WeaponFactsSourceCatalog = "catalog"
	// WeaponFactsSourceAdapter is the adapter.yaml of a package added with
	// `provider add` and bound with mode custom.
	WeaponFactsSourceAdapter = "adapter"
)

// ErrWeaponFactsNotFound means neither the catalog nor a bound custom package
// describes the provider.
var ErrWeaponFactsNotFound = errors.New("weapon facts not found")

// ErrWeaponFactsAmbiguous means a plain id names several catalogued versions;
// the reference must be written id@version (ADR-0061 Decision 9).
var ErrWeaponFactsAmbiguous = errors.New("weapon reference is ambiguous across versions")

// WeaponFacts is the risk, role and runtime facts preflight and role
// validation need about one Weapon, independent of where they were read from.
type WeaponFacts struct {
	ID string
	// Version is the catalogued or adapter-declared version. Together with ID it
	// names the skills/<id>@<version>/ payload when the Weapon has one.
	Version        string
	Source         string
	RiskScore      string
	CanonicalRole  string
	Roles          []string
	ScratchRoot    string
	WeaponContract WeaponContract
	// CompatibilitySource is the catalog classification (native_role, embedded,
	// external).
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
	// Entrypoints are the entrypoints an adapter.yaml declares; catalog entries do
	// not declare them here.
	Entrypoints []string
}

// WeaponFactsDocument is the subset of a catalog entry that WeaponFacts reads.
// The YAML tags describe the boundary document; decoding remains owned by the
// catalog/filesystem adapter, not by domain.
type WeaponFactsDocument struct {
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

// WeaponFacts converts a decoded boundary document into domain facts.
func (d WeaponFactsDocument) WeaponFacts(source string) WeaponFacts {
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

// CatalogDocumentsForReference returns the catalog entries named by ref:
// every version for a plain id, or the one version for an id@version ref.
func CatalogDocumentsForReference(docs []WeaponFactsDocument, ref string) []WeaponFactsDocument {
	id, version := ParseWeaponRef(ref)
	var matches []WeaponFactsDocument
	for _, entry := range docs {
		if entry.ID == id && (version == "" || CatalogDocumentVersion(entry) == version) {
			matches = append(matches, entry)
		}
	}
	return matches
}

// CatalogDocumentVersion returns the effective catalog version used by
// identity matching. Legacy entries without a version are version 1.0.0.
func CatalogDocumentVersion(doc WeaponFactsDocument) string {
	if doc.Version == "" {
		return DefaultWeaponVersion
	}
	return doc.Version
}

// AmbiguousWeaponFactsError creates the stable error returned when an id names
// multiple catalog versions.
func AmbiguousWeaponFactsError(ref string, matches []WeaponFactsDocument) error {
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, WeaponIdentity(match.ID, CatalogDocumentVersion(match)))
	}
	return fmt.Errorf("%w: %q is catalogued as %s", ErrWeaponFactsAmbiguous, ref, strings.Join(names, ", "))
}

// WeaponFactsFromAdapter converts an already decoded custom adapter document
// into domain facts. YAML and filesystem access stay outside domain.
func WeaponFactsFromAdapter(provider string, adapter AdapterContract) WeaponFacts {
	facts := WeaponFacts{
		ID: provider, Source: WeaponFactsSourceAdapter, RiskScore: adapter.RiskScore, Roles: adapter.SupportedRoles,
		ScratchRoot: adapter.ScratchRoot, SupportedSlots: adapter.SupportedSlots, RequestedPermissions: adapter.RequestedPermissions,
		Entrypoints: adapter.Entrypoints,
	}
	if len(adapter.SupportedRoles) > 0 {
		facts.CanonicalRole = adapter.SupportedRoles[0]
	}
	return facts
}

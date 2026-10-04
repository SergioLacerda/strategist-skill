package domain

import (
	"fmt"
	"sort"
	"strings"
)

// Identity is the Weapon's registry identity, "id@version".
func (w CompiledWeapon) Identity() string { return WeaponIdentity(w.ID, w.Version) }

// WeaponIdentity renders the versioned registry identity.
func WeaponIdentity(id, version string) string { return id + "@" + version }

// WeaponRef renders a versioned reference, or the bare id when unversioned.
func WeaponRef(id, version string) string {
	if version == "" {
		return id
	}
	return WeaponIdentity(id, version)
}

// DefaultWeaponVersion is used when a legacy catalog entry declares no version.
const DefaultWeaponVersion = "0.0.0"

// WeaponPayloadDirName returns the flat skills directory name for one version.
func WeaponPayloadDirName(id, version string) string {
	if version == "" {
		version = DefaultWeaponVersion
	}
	return WeaponIdentity(id, version)
}

// Weapon returns the unique compiled Weapon for an id and version.
func (r CompiledRegistry) Weapon(id, version string) (CompiledWeapon, bool) {
	return findCompiledWeapon(r.Weapons, id, version)
}

// Role returns the unique compiled Role for an id.
func (r CompiledRegistry) Role(id string) (CompiledRole, bool) { return findCompiledRole(r.Roles, id) }

// RankedBinding returns the certified binding for the complete Weapon identity.
func (r CompiledRegistry) RankedBinding(role, slot, weaponID, weaponVersion string) (CompiledRankedBinding, bool) {
	for _, b := range r.RankedBindings {
		if b.Role == role && b.Slot == slot && b.WeaponID == weaponID && b.WeaponVersion == weaponVersion && weaponVersion != "" {
			return b, true
		}
	}
	return CompiledRankedBinding{}, false
}

// RankedBindingsFor lists the certified offers for a Role slot in stable order.
func (r CompiledRegistry) RankedBindingsFor(role, slot string) []CompiledRankedBinding {
	var offers []CompiledRankedBinding
	for _, b := range r.RankedBindings {
		if b.Role == role && b.Slot == slot {
			offers = append(offers, b)
		}
	}
	sort.Slice(offers, func(i, j int) bool {
		if offers[i].WeaponID != offers[j].WeaponID {
			return offers[i].WeaponID < offers[j].WeaponID
		}
		return CompareWeaponVersions(offers[i].WeaponVersion, offers[j].WeaponVersion) < 0
	})
	return offers
}

// ParseWeaponRef splits an id@version reference, retaining an empty version.
func ParseWeaponRef(ref string) (id, version string) {
	id, version, _ = strings.Cut(ref, "@")
	return id, version
}

// CompiledRegistryDocument is the decoded registry section of catalog.yaml.
// YAML decoding is owned by internal/catalog; this type and its validation are
// kept in the domain so runtime adapters cannot invent a second contract.
type CompiledRegistryDocument struct {
	SchemaVersion   string                  `yaml:"schema_version"`
	TaxonomyVersion string                  `yaml:"taxonomy_version"`
	Weapons         []CompiledWeapon        `yaml:"weapons"`
	Roles           []CompiledRole          `yaml:"roles"`
	Compatibility   []CompiledCompatibility `yaml:"compatibility"`
	RankedBindings  []CompiledRankedBinding `yaml:"ranked_bindings"`
}

// CompiledRegistryFromDocument validates and materializes a decoded registry.
func CompiledRegistryFromDocument(e CompiledRegistryDocument) (CompiledRegistry, error) {
	if e.SchemaVersion == "" {
		return CompiledRegistry{}, fmt.Errorf("parse compiled registry catalog: catalog schema_version is required")
	}
	taxonomyVersion := e.TaxonomyVersion
	if taxonomyVersion == "" {
		taxonomyVersion = CanonicalTaxonomyVersion
	}
	r := CompiledRegistry{SchemaVersion: CompiledRegistrySchemaVersion, TaxonomyVersion: taxonomyVersion, Weapons: e.Weapons, Roles: e.Roles, Compatibility: e.Compatibility, RankedBindings: e.RankedBindings}
	if err := r.Validate(); err != nil {
		return CompiledRegistry{}, err
	}
	return r, nil
}

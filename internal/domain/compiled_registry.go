package domain

import (
	"fmt"
)

// CompiledRegistrySchemaVersion identifies the normalized registry embedded
// in the CLI. It is generated from the catalog and native Role contracts.
const CompiledRegistrySchemaVersion = "strategist-compiled-role-weapon-registry/v1"

// CompiledRegistry is the build-time immutable view of the internal Weapon,
// Role, and Ranked binding graph.
type CompiledRegistry struct {
	SchemaVersion  string                  `yaml:"schema_version"`
	Weapons        []CompiledWeapon        `yaml:"weapons"`
	Roles          []CompiledRole          `yaml:"roles"`
	Compatibility  []CompiledCompatibility `yaml:"compatibility,omitempty"`
	RankedBindings []CompiledRankedBinding `yaml:"ranked_bindings"`
}

// CompiledWeapon is the normalized build identity of one Weapon.
type CompiledWeapon struct {
	ID             string        `yaml:"id"`
	Version        string        `yaml:"version,omitempty"`
	Digest         string        `yaml:"digest"`
	SourceDigest   string        `yaml:"source_digest,omitempty"`
	Origin         WeaponOrigin  `yaml:"origin"`
	Runtime        WeaponRuntime `yaml:"runtime"`
	ConnectorID    string        `yaml:"connector_id,omitempty"`
	Entrypoint     string        `yaml:"entrypoint,omitempty"`
	SupportedRoles []string      `yaml:"supported_roles,omitempty"`
	SupportedSlots []string      `yaml:"supported_slots,omitempty"`
}

// CompiledCompatibility is one build-derived Role/Slot/Weapon option. It is
// distinct from Ranked binding: compatibility describes what is possible,
// while Ranked binding describes what was certified and fixed at build time.
type CompiledCompatibility struct {
	Role          string `yaml:"role"`
	Slot          string `yaml:"slot"`
	WeaponID      string `yaml:"weapon_id"`
	WeaponVersion string `yaml:"weapon_version,omitempty"`
	WeaponDigest  string `yaml:"weapon_digest"`
	HandoffSchema string `yaml:"handoff_schema,omitempty"`
	Source        string `yaml:"source"`
}

// CompiledRole is the normalized build identity of one active pipeline Role.
type CompiledRole struct {
	ID             string `yaml:"id"`
	Slot           string `yaml:"slot"`
	ContractDigest string `yaml:"contract_digest"`
	HandoffSchema  string `yaml:"handoff_schema,omitempty"`
}

// CompiledRankedBinding is an immutable, build-certified Role→Weapon link.
type CompiledRankedBinding struct {
	Role                string        `yaml:"role"`
	Slot                string        `yaml:"slot"`
	WeaponID            string        `yaml:"weapon_id"`
	WeaponVersion       string        `yaml:"weapon_version"`
	WeaponDigest        string        `yaml:"weapon_digest"`
	RoleDigest          string        `yaml:"role_digest"`
	BindingDigest       string        `yaml:"binding_digest"`
	SourceDigest        string        `yaml:"source_digest,omitempty"`
	ExecutionMode       string        `yaml:"execution_mode"`
	CertificationDigest string        `yaml:"certification_digest"`
	ConnectorID         string        `yaml:"connector_id"`
	Runtime             WeaponRuntime `yaml:"runtime"`
	Entrypoint          string        `yaml:"entrypoint"`
	Generation          int64         `yaml:"generation"`
	Status              string        `yaml:"status"`
}

// Validate verifies the complete compiled graph before it is embedded or
// consumed by install/runtime code.
func (r CompiledRegistry) Validate() error {
	if r.SchemaVersion != CompiledRegistrySchemaVersion {
		return fmt.Errorf("compiled registry: unsupported schema %q", r.SchemaVersion)
	}
	if len(r.Weapons) == 0 {
		return fmt.Errorf("compiled registry: weapons must not be empty")
	}
	if len(r.Roles) == 0 {
		return fmt.Errorf("compiled registry: roles must not be empty")
	}
	if err := validateCompiledWeapons(r.Weapons); err != nil {
		return err
	}
	if err := validateCompiledRoles(r.Roles); err != nil {
		return err
	}
	if err := validateCompiledCompatibility(r.Weapons, r.Roles, r.Compatibility); err != nil {
		return err
	}
	return validateCompiledRankedBindings(r.Weapons, r.Roles, r.RankedBindings)
}

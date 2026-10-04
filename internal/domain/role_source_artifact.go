package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// RoleSourceArtifactSchemaVersion identifies the source/projection identity
// for one built-in Role. It records the two authoritative inputs that are
// materialized together: the Role directives and its native skill manifest.
const RoleSourceArtifactSchemaVersion = "strategist-role-source/v1"

// RoleSourceArtifact keeps Role contract identity separate from runtime
// readiness and Weapon binding. Digests are raw SHA-256 hex values of the
// canonical embedded files.
type RoleSourceArtifact struct {
	SchemaVersion       string `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion     string `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Role                string `json:"role" yaml:"role"`
	RoleConfigPath      string `json:"role_config_path" yaml:"role_config_path"`
	SkillManifestPath   string `json:"skill_manifest_path" yaml:"skill_manifest_path"`
	RoleConfigDigest    string `json:"role_config_digest" yaml:"role_config_digest"`
	SkillManifestDigest string `json:"skill_manifest_digest" yaml:"skill_manifest_digest"`
	SkillSchemaVersion  string `json:"skill_schema_version" yaml:"skill_schema_version"`
	SkillVersion        string `json:"skill_version" yaml:"skill_version"`
	HandoffSchema       string `json:"handoff_schema,omitempty" yaml:"handoff_schema,omitempty"`
}

// NewRoleSourceArtifact creates and validates a source identity projection.
func NewRoleSourceArtifact(role, skillSchemaVersion, skillVersion, handoffSchema, roleDigest, skillDigest string) (RoleSourceArtifact, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	artifact := RoleSourceArtifact{
		SchemaVersion:       RoleSourceArtifactSchemaVersion,
		TaxonomyVersion:     CanonicalTaxonomyVersion,
		Role:                role,
		RoleConfigPath:      "roles/" + role + ".yaml",
		SkillManifestPath:   "internal_skills/" + role + "/skill.yaml",
		RoleConfigDigest:    strings.TrimSpace(roleDigest),
		SkillManifestDigest: strings.TrimSpace(skillDigest),
		SkillSchemaVersion:  strings.TrimSpace(skillSchemaVersion),
		SkillVersion:        strings.TrimSpace(skillVersion),
		HandoffSchema:       strings.TrimSpace(handoffSchema),
	}
	if err := artifact.Validate(); err != nil {
		return RoleSourceArtifact{}, err
	}
	return artifact, nil
}

// Validate checks source paths, role identity, manifest identity, and digests.
func (a RoleSourceArtifact) Validate() error {
	if a.SchemaVersion != RoleSourceArtifactSchemaVersion {
		return fmt.Errorf("role source artifact: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("role source artifact: %w", err)
	}
	if err := validateRoleSourceIdentity(a); err != nil {
		return err
	}
	if err := validateRoleSourcePaths(a); err != nil {
		return err
	}
	if err := validateRoleSourceDigests(a); err != nil {
		return err
	}
	return validateRoleSkillIdentity(a)
}

func validateRoleSourceIdentity(a RoleSourceArtifact) error {
	if err := ValidateRoleReference(a.Role); err != nil {
		return fmt.Errorf("role source artifact: %w", err)
	}
	return nil
}

func validateRoleSourcePaths(a RoleSourceArtifact) error {
	if a.RoleConfigPath != "roles/"+a.Role+".yaml" {
		return fmt.Errorf("role source artifact: role_config_path does not match role %q", a.Role)
	}
	if a.SkillManifestPath != "internal_skills/"+a.Role+"/skill.yaml" {
		return fmt.Errorf("role source artifact: skill_manifest_path does not match role %q", a.Role)
	}
	return nil
}

func validateRoleSourceDigests(a RoleSourceArtifact) error {
	if err := validateRoleSourceDigest("role_config_digest", a.RoleConfigDigest); err != nil {
		return err
	}
	return validateRoleSourceDigest("skill_manifest_digest", a.SkillManifestDigest)
}

func validateRoleSkillIdentity(a RoleSourceArtifact) error {
	if strings.TrimSpace(a.SkillSchemaVersion) == "" || strings.TrimSpace(a.SkillVersion) == "" {
		return fmt.Errorf("role source artifact: skill manifest identity is required")
	}
	return nil
}

func validateRoleSourceDigest(field, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("role source artifact: %s must be a SHA-256 hex digest", field)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("role source artifact: %s must be a SHA-256 hex digest: %w", field, err)
	}
	return nil
}

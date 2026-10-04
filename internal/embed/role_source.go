package embed

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

type roleSkillManifest struct {
	ID            string `yaml:"id"`
	Version       string `yaml:"version"`
	SchemaVersion string `yaml:"schema_version"`
}

// RoleSourceArtifact reads the canonical embedded Role directives and native
// skill manifest, returning one identity artifact for both projections.
func (e Extractor) RoleSourceArtifact(roleID string) (domain.RoleSourceArtifact, error) {
	roleID = strings.ToLower(strings.TrimSpace(roleID))
	if unsafePayloadSegment(roleID) || strings.Contains(roleID, "@") {
		return domain.RoleSourceArtifact{}, fmt.Errorf("embed: invalid role id %q", roleID)
	}
	roleRaw, role, err := e.readRoleSource(roleID)
	if err != nil {
		return domain.RoleSourceArtifact{}, err
	}
	skillRaw, skill, err := e.readRoleSkillManifest(roleID)
	if err != nil {
		return domain.RoleSourceArtifact{}, err
	}
	if strings.ToLower(strings.TrimSpace(skill.ID)) != roleID {
		return domain.RoleSourceArtifact{}, fmt.Errorf("embed: internal_skills/%s/skill.yaml declares id %q, expected %q", roleID, skill.ID, roleID)
	}
	artifact, err := domain.NewRoleSourceArtifact(
		roleID, skill.SchemaVersion, skill.Version, role.HandoffSchema,
		domain.SHA256Hex(roleRaw), domain.SHA256Hex(skillRaw),
	)
	if err != nil {
		return domain.RoleSourceArtifact{}, fmt.Errorf("embed: build role source artifact: %w", err)
	}
	return artifact, nil
}

func (e Extractor) readRoleSource(roleID string) ([]byte, domain.RoleConfig, error) {
	path := "roles/" + roleID + ".yaml"
	raw, err := e.ReadFile(path)
	if err != nil {
		return nil, domain.RoleConfig{}, err
	}
	var role domain.RoleConfig
	if err := yaml.Unmarshal(raw, &role); err != nil {
		return nil, domain.RoleConfig{}, fmt.Errorf("embed: parse %s: %w", path, err)
	}
	if err := role.Validate(); err != nil {
		return nil, domain.RoleConfig{}, fmt.Errorf("embed: validate %s: %w", path, err)
	}
	if strings.ToLower(strings.TrimSpace(role.Role)) != roleID {
		return nil, domain.RoleConfig{}, fmt.Errorf("embed: %s declares role %q, expected %q", path, role.Role, roleID)
	}
	return raw, role, nil
}

func (e Extractor) readRoleSkillManifest(roleID string) ([]byte, roleSkillManifest, error) {
	path := "internal_skills/" + roleID + "/skill.yaml"
	raw, err := e.ReadFile(path)
	if err != nil {
		return nil, roleSkillManifest{}, err
	}
	manifest, err := parseRoleSkillManifest(raw)
	if err != nil {
		return nil, roleSkillManifest{}, fmt.Errorf("embed: parse %s: %w", path, err)
	}
	return raw, manifest, nil
}

// parseRoleSkillManifest reads only the stable identity header. The remainder
// of native skill.yaml is agent-facing contract text and is intentionally not
// required to be accepted as strict YAML during this compatibility migration.
func parseRoleSkillManifest(raw []byte) (roleSkillManifest, error) {
	var manifest roleSkillManifest
	for _, line := range strings.Split(string(raw), "\n") {
		updateRoleSkillManifest(&manifest, line)
	}
	if strings.TrimSpace(manifest.ID) == "" || strings.TrimSpace(manifest.Version) == "" || strings.TrimSpace(manifest.SchemaVersion) == "" {
		return roleSkillManifest{}, fmt.Errorf("identity header requires id, version, and schema_version")
	}
	return manifest, nil
}

func updateRoleSkillManifest(manifest *roleSkillManifest, line string) {
	key, value, ok := strings.Cut(line, ":")
	if !ok || strings.TrimSpace(key) != key {
		return
	}
	value = strings.Trim(strings.TrimSpace(value), "\"")
	switch strings.TrimSpace(key) {
	case "id":
		manifest.ID = value
	case "version":
		manifest.Version = value
	case "schema_version":
		manifest.SchemaVersion = value
	}
}

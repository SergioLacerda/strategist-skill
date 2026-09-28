package install

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func catalogProviderFromIngestedSkill(skill IngestedSkill) pluginCatalogProvider {
	return pluginCatalogProvider{
		ID:                      skill.ID,
		Version:                 skill.Package.Version,
		SchemaVersion:           "1",
		Kind:                    skill.Adapter.Kind,
		Origin:                  domain.WeaponOriginEmbedded,
		Status:                  "active",
		RiskScore:               skill.Adapter.RiskScore,
		Category:                skill.Adapter.Category,
		CanonicalRole:           skill.Adapter.CanonicalRole,
		Roles:                   skill.Adapter.Roles,
		Lifecycle:               skill.Adapter.Lifecycle,
		Default:                 skill.Adapter.Default,
		Description:             skillDescription(skill.Dir),
		AuxiliaryTools:          skill.Adapter.AuxiliaryTools,
		Installable:             true,
		LegacyManifestPath:      "skills/" + skill.ID + "/skill.yaml",
		CompatibilitySource:     "embedded",
		SupportedHandoffSchemas: skill.Adapter.SupportedHandoffSchemas,
		ScratchRoot:             skill.Adapter.ScratchRoot,
		WeaponContract:          skill.Adapter.WeaponContract,
		Runtime:                 skill.Adapter.Runtime,
		SupportedSlots:          skill.Adapter.SupportedSlots,
		Composition:             skill.Adapter.Composition,
		UpstreamRepo:            skill.Adapter.UpstreamRepo,
		UpstreamSkillPath:       skill.Adapter.UpstreamSkillPath,
		UpstreamVersion:         skill.Adapter.UpstreamVersion,
		UpstreamCommit:          skill.Adapter.UpstreamCommit,
		UpstreamContentDigest:   skill.Adapter.UpstreamContentDigest,
		License:                 skill.Adapter.License,
	}
}

// skillDescription reads the ORKA-portable "description" frontmatter field
// straight from SKILL.md for the generated catalog entry — a thin metadata
// mirror only, never vendoring the package's own prose body (ADR-0029
// DEC-001). Returns an empty string if SKILL.md is missing or malformed;
// resolveExternalSkill already rejects that case earlier in the pipeline, so
// this is a defensive fallback, not the primary validation path.
func skillDescription(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md")) //nolint:gosec // G304: dir is an operator-declared ingestion source, not untrusted request input
	if err != nil {
		return ""
	}
	return parseSkillDescription(string(raw))
}

func parseSkillDescription(content string) string {
	var fm struct {
		Description string `yaml:"description"`
	}
	const delim = "---"
	if !strings.HasPrefix(content, delim) {
		return ""
	}
	rest := content[len(delim):]
	end := strings.Index(rest, "\n"+delim)
	if end < 0 {
		return ""
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return ""
	}
	return fm.Description
}

package install

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func catalogProviderFromIngestedSkill(skill IngestedSkill) pluginCatalogProvider {
	return pluginCatalogProvider{
		ID:                  skill.ID,
		Version:             skill.Package.Version,
		SchemaVersion:       "1",
		Status:              "active",
		RiskScore:           skill.Adapter.RiskScore,
		Category:            skill.Adapter.Category,
		CanonicalRole:       skill.Adapter.CanonicalRole,
		Default:             skill.Adapter.Default,
		Description:         skillDescription(skill.Dir),
		AuxiliaryTools:      skill.Adapter.AuxiliaryTools,
		Installable:         true,
		LegacyManifestPath:  "skills/" + skill.ID + "/skill.yaml",
		CompatibilitySource: "embedded",
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
	var fm struct {
		Description string `yaml:"description"`
	}
	content := string(raw)
	const delim = "---"
	if len(content) < len(delim) || content[:len(delim)] != delim {
		return ""
	}
	rest := content[len(delim):]
	end := -1
	needle := "\n" + delim
	for i := 0; i+len(needle) <= len(rest); i++ {
		if rest[i:i+len(needle)] == needle {
			end = i
			break
		}
	}
	if end < 0 {
		return ""
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return ""
	}
	return fm.Description
}

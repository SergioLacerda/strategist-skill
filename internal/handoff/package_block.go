package handoff

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadPackageFrontmatterBlock decodes one typed block of a refined package's
// analysis.md frontmatter into out. found is false when the package declares no
// such block: absence is reported, never invented.
func ReadPackageFrontmatterBlock(refined, key string, out any) (found bool, err error) {
	content, err := readArtifact(filepath.Join(refined, "analysis.md"))
	if err != nil {
		return false, fmt.Errorf("handoff_artifact_invalid: read Archivist analysis: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(content)
	if err != nil {
		return false, fmt.Errorf("handoff_artifact_invalid: Archivist analysis: %w", err)
	}
	raw, present := frontmatter[key]
	if !present || raw == nil {
		return false, nil
	}
	encoded, err := yaml.Marshal(raw)
	if err != nil {
		return true, fmt.Errorf("handoff_artifact_invalid: encode %s: %w", key, err)
	}
	if err := yaml.Unmarshal(encoded, out); err != nil {
		return true, fmt.Errorf("handoff_artifact_invalid: decode %s: %w", key, err)
	}
	return true, nil
}

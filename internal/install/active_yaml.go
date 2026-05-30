// Package install orchestrates the Strategist skill installation into a target repository.
package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// writeActiveYAML writes active.yaml to strategistDir from wizard config values.
// In silent mode (no wizard), the extract step already copied the template
// active.yaml from defaults, so nothing extra is needed.
func writeActiveYAML(strategistDir string, wc domain.WizardConfig) error {
	adrEnabled := "true"
	if !wc.AdrEnabled {
		adrEnabled = "false"
	}

	content := fmt.Sprintf(`mode: %s
base_path: %s
knowledge_index_path: knowledge.index.yaml
language: %s
adr_enabled: %s

slots:
  discovery: %s
  refinement: %s
  execution: %s
`,
		wc.Mode, wc.BasePath, wc.Language, adrEnabled,
		wc.DiscoveryProvider, wc.RefinementProvider, wc.ExecutionProvider,
	)

	if wc.TreasureChestPath != "" {
		id := treasureChestID(wc.TreasureChestPath)
		content += fmt.Sprintf(`
treasure_chests:
  - id: %s
    path: %s
    scope: all
`, id, wc.TreasureChestPath)
	}

	path := filepath.Join(strategistDir, "active.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write active.yaml: %w", err)
	}
	return nil
}

// treasureChestID derives a stable id from a path by taking the last non-empty segment.
func treasureChestID(path string) string {
	path = strings.TrimRight(path, "/")
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

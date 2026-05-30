// Package install orchestrates the Strategist skill installation into a target repository.
package install

import (
	"fmt"
	"os"
	"path/filepath"

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

	path := filepath.Join(strategistDir, "active.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write active.yaml: %w", err)
	}
	return nil
}

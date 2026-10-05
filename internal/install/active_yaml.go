// Package install orchestrates the Strategist skill installation into a target repository.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
)

// writeActiveYAML writes active.yaml to strategistDir from wizard config values.
// In silent mode (no wizard), the extract step already copied the selected template
// active.yaml from defaults, so nothing extra is needed.
//
// Every field emitted here must come from wc — this function fully regenerates
// active.yaml from scratch on each call (no read-merge of an existing file), matching
// how the rest of the wizard flow already behaves for every other field.
func writeActiveYAML(strategistDir string, wc domain.WizardConfig) error {
	if err := validateDocumentationPaths(wc); err != nil {
		return err
	}
	content := activeYAMLCore(wc)
	content += documentationRootsYAML(wc.DocumentationRoots)
	content += treasureChestYAML(wc.TreasureChestPath)
	content += canonicalPathsYAML(wc)
	levelingBlock, err := levelingActiveYAML(wc.Leveling)
	if err != nil {
		return err
	}
	content += levelingBlock
	return writeActiveYAMLBytes(strategistDir, []byte(content))
}

func validateDocumentationPaths(wc domain.WizardConfig) error {
	if err := domain.ValidateProjectRelativeDirectory(wc.AdrCanonicalPath, "adr.canonical_path"); err != nil {
		return fmt.Errorf("validate adr.canonical_path: %w", err)
	}
	if err := domain.ValidateProjectRelativeDirectory(wc.RunbookCanonicalPath, "runbooks.canonical_path"); err != nil {
		return fmt.Errorf("validate runbooks.canonical_path: %w", err)
	}
	return nil
}

func activeYAMLCore(wc domain.WizardConfig) string {
	return fmt.Sprintf(`mode: %s
base_path: %s
knowledge_index_path: knowledge.index.yaml
language:
  ui: %s
  docs: %s
  chat: %s
  code: %s

slots:
  discovery: %s
  refinement: %s
  execution: %s
`,
		wc.Mode, wc.BasePath,
		wc.UILanguage, wc.DocLanguage, wc.ChatLanguage, wc.CodeLanguage,
		wc.DiscoveryProvider, wc.RefinementProvider, wc.ExecutionProvider,
	)
}

func documentationRootsYAML(roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	content := "documentation_roots:\n"
	for _, root := range roots {
		content += fmt.Sprintf("  - %s\n", root)
	}
	return content
}

func treasureChestYAML(path string) string {
	if path == "" {
		return ""
	}
	id := treasureChestID(path)
	return fmt.Sprintf(`
treasure_chests:
  - id: %s
    path: %s
    scope: all
`, id, path)
}

func canonicalPathsYAML(wc domain.WizardConfig) string {
	content := ""
	if wc.AdrCanonicalPath != "" {
		content += fmt.Sprintf(`
adr:
  canonical_path: %s
`, wc.AdrCanonicalPath)
	}
	if wc.RunbookCanonicalPath != "" {
		content += fmt.Sprintf(`
runbooks:
  canonical_path: %s
`, wc.RunbookCanonicalPath)
	}
	return content
}

// writeActiveYAMLBytes atomically writes active.yaml under strategistDir and
// seals .config.lock over it. Both silent and wizard installs write active.yaml
// through this path so neither can start without an integrity lock.
func writeActiveYAMLBytes(strategistDir string, data []byte) error {
	path := filepath.Join(strategistDir, activeYAMLName)
	if err := atomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write active.yaml: %w", err)
	}
	lockPath := filepath.Join(strategistDir, configLockName)
	if err := integrity.WriteLock(path, lockPath); err != nil {
		fmt.Fprintf(os.Stderr, "[Strategist] WARN: could not write config lock: %v\n", err)
	}
	return nil
}

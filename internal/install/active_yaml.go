// Package install orchestrates the Strategist skill installation into a target repository.
package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
)

// writeActiveYAML writes active.yaml to strategistDir from wizard config values.
// In silent mode (no wizard), the extract step already copied the selected template
// active.yaml from defaults, so nothing extra is needed.
func writeActiveYAML(strategistDir string, wc domain.WizardConfig) error {
	content := fmt.Sprintf(`mode: %s
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

	if wc.TreasureChestPath != "" {
		id := treasureChestID(wc.TreasureChestPath)
		content += fmt.Sprintf(`
treasure_chests:
  - id: %s
    path: %s
    scope: all
`, id, wc.TreasureChestPath)
	}

	return writeActiveYAMLBytes(strategistDir, []byte(content))
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

// writeKnowledgeIndexSource updates knowledge.index.yaml to include a source entry
// for the wizard's TreasureChestPath. It replaces the "sources: []" placeholder
// in the extracted template, preserving surrounding comments.
// No-op when TreasureChestPath is empty.
func writeKnowledgeIndexSource(strategistDir string, wc domain.WizardConfig) error {
	if wc.TreasureChestPath == "" {
		return nil
	}
	kiPath := filepath.Join(strategistDir, knowledgeIndexName)
	data, err := os.ReadFile(kiPath) //nolint:gosec // G304: path constructed from install config
	if err != nil {
		return fmt.Errorf("read knowledge.index.yaml: %w", err)
	}
	id := treasureChestID(wc.TreasureChestPath)
	entry := fmt.Sprintf("sources:\n  - id: %s\n    path: %s\n    tags: [all]", id, wc.TreasureChestPath)
	updated, err := replacePlaceholder(string(data), "sources: []", entry)
	if err != nil {
		return fmt.Errorf("knowledge.index.yaml: %w", err)
	}
	if err := atomicWriteFile(kiPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write knowledge.index.yaml: %w", err)
	}
	return nil
}

// writeTreasureChestManifest populates the chests: [] placeholder in treasure-chests.yaml
// with a T1 chest entry for the wizard's TreasureChestPath. No-op when path is empty.
func writeTreasureChestManifest(strategistDir string, wc domain.WizardConfig) error {
	if wc.TreasureChestPath == "" {
		return nil
	}
	tcPath := filepath.Join(strategistDir, treasureChestsName)
	data, err := os.ReadFile(tcPath) //nolint:gosec // G304: path constructed from install config
	if err != nil {
		return fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	id := treasureChestID(wc.TreasureChestPath)
	entry := fmt.Sprintf(`chests:
  - id: %s
    title: %s
    path: %s
    trust:
      tier: T1
      reviewed_by: human
    routing:
      task_types: [all]
    retrieval:
      strategy: selective
      require_relevance_reason: true
      allow_full_load: false`, id, id, wc.TreasureChestPath)
	updated, err := replacePlaceholder(string(data), "chests: []", entry)
	if err != nil {
		return fmt.Errorf("treasure-chests.yaml: %w", err)
	}
	if err := atomicWriteFile(tcPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write treasure-chests.yaml: %w", err)
	}
	return nil
}

// replacePlaceholder replaces the first occurrence of placeholder in content with
// replacement, returning an explicit error if placeholder is absent. This guards
// template mutation against silently no-op-ing when template formatting drifts.
func replacePlaceholder(content, placeholder, replacement string) (string, error) {
	if !strings.Contains(content, placeholder) {
		return "", fmt.Errorf("placeholder %q not found", placeholder)
	}
	return strings.Replace(content, placeholder, replacement, 1), nil
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

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
//
// Every field emitted here must come from wc — this function fully regenerates
// active.yaml from scratch on each call (no read-merge of an existing file), matching
// how the rest of the wizard flow already behaves for every other field.
// wc.AdrCanonicalPath is not currently populated by the interactive wizard (see
// TestWizardDoesNotAskPermissionLevel), so it is a no-op unless a caller constructs
// WizardConfig programmatically.
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

	if wc.AdrCanonicalPath != "" {
		content += fmt.Sprintf(`
adr:
  canonical_path: %s
`, wc.AdrCanonicalPath)
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
	updated, changed, err := substituteOrSkip(string(data), "sources: []", "sources:", entry)
	if err != nil {
		return fmt.Errorf("knowledge.index.yaml: %w", err)
	}
	if !changed {
		return nil // already configured by a prior wizard run — not an error
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
	updated, changed, err := substituteOrSkip(string(data), "chests: []", "chests:", entry)
	if err != nil {
		return fmt.Errorf("treasure-chests.yaml: %w", err)
	}
	if !changed {
		return nil // already configured by a prior wizard run — not an error
	}
	if err := atomicWriteFile(tcPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write treasure-chests.yaml: %w", err)
	}
	return nil
}

// substituteOrSkip replaces the first occurrence of the one-liner placeholder
// (e.g. "sources: []") with replacement. If the placeholder is absent but key
// (e.g. "sources:") is present, the target was already substituted by a prior
// install run — that is a legitimate re-run, not template drift, so it returns
// the content unchanged with changed=false and no error. Only when neither the
// placeholder nor the key is found does it report an error: that combination
// means the template is missing the section entirely, which is genuine
// corruption/drift this guard exists to catch.
func substituteOrSkip(content, placeholder, key, replacement string) (updated string, changed bool, err error) {
	if strings.Contains(content, placeholder) {
		return strings.Replace(content, placeholder, replacement, 1), true, nil
	}
	if strings.Contains(content, key) {
		return content, false, nil
	}
	return "", false, fmt.Errorf("placeholder %q not found and %q key absent — template may be corrupted", placeholder, key)
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

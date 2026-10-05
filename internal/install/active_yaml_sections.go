package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

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
		return nil
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
		return nil
	}
	if err := atomicWriteFile(tcPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write treasure-chests.yaml: %w", err)
	}
	return nil
}

// substituteOrSkip replaces the first occurrence of a one-line placeholder.
func substituteOrSkip(content, placeholder, key, replacement string) (string, bool, error) {
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

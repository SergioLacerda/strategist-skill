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
// In silent mode (no wizard), the extract step already copied the selected template
// active.yaml from defaults, so nothing extra is needed.
func writeActiveYAML(strategistDir string, wc domain.WizardConfig) error {
	missionMode := wc.MissionMode
	if missionMode == "" {
		missionMode = domain.MissionModeFromLegacy(wc.DoneScope, wc.ApplyChanges)
	}
	policy := domain.NewMissionPolicy(missionMode)

	adrEnabled := "true"
	if !wc.AdrEnabled {
		adrEnabled = "false"
	}
	applyChanges := "false"
	if policy.ApplyChanges {
		applyChanges = "true"
	}

	content := fmt.Sprintf(`mode: %s
base_path: %s
roles_config: roles/default.yaml
knowledge_index_path: knowledge.index.yaml
language:
  ui: %s
  docs: %s
  chat: %s
  code: %s
adr_enabled: %s
mission_mode: %s
escopo_done: %s
aplicar_alteracoes: %s

slots:
  discovery: %s
  refinement: %s
  execution: %s
`,
		wc.Mode, wc.BasePath,
		wc.UILanguage, wc.DocLanguage, wc.ChatLanguage, wc.CodeLanguage,
		adrEnabled,
		policy.Mode,
		policy.DoneScope,
		applyChanges,
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

// writeKnowledgeIndexSource updates knowledge.index.yaml to include a source entry
// for the wizard's TreasureChestPath. It replaces the "sources: []" placeholder
// in the extracted template, preserving surrounding comments.
// No-op when TreasureChestPath is empty.
func writeKnowledgeIndexSource(strategistDir string, wc domain.WizardConfig) error {
	if wc.TreasureChestPath == "" {
		return nil
	}
	kiPath := filepath.Join(strategistDir, "knowledge.index.yaml")
	data, err := os.ReadFile(kiPath) //nolint:gosec // G304: path constructed from install config
	if err != nil {
		return fmt.Errorf("read knowledge.index.yaml: %w", err)
	}
	id := treasureChestID(wc.TreasureChestPath)
	entry := fmt.Sprintf("sources:\n  - id: %s\n    path: %s\n    tags: [all]", id, wc.TreasureChestPath)
	updated := strings.Replace(string(data), "sources: []", entry, 1)
	if err := os.WriteFile(kiPath, []byte(updated), 0o644); err != nil { //nolint:gosec // G703: path from install config
		return fmt.Errorf("write knowledge.index.yaml: %w", err)
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

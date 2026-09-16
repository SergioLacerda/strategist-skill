package treasure

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type activeYAMLWithChests struct {
	TreasureChests []ActiveChest `yaml:"treasure_chests"`
}

type indexedEntry struct {
	ID string `yaml:"id"`
}

type knowledgeIndexYAML struct {
	Sources []indexedEntry `yaml:"sources"`
}

// LoadActiveChests reads active.yaml treasure_chests entries.
func LoadActiveChests(root string) ([]ActiveChest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // G304: active.yaml path is derived from the selected runtime root
	if err != nil {
		return nil, fmt.Errorf("read active.yaml: %w", err)
	}
	var cfg activeYAMLWithChests
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse active.yaml: %w", err)
	}
	return cfg.TreasureChests, nil
}

// LoadIndexed reads knowledge.index.yaml and returns indexed source ids.
func LoadIndexed(root string) (map[string]bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "knowledge.index.yaml")) //nolint:gosec // G304: knowledge index path is derived from the selected runtime root
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read knowledge.index.yaml: %w", err)
	}
	var ki knowledgeIndexYAML
	if err := yaml.Unmarshal(raw, &ki); err != nil {
		return nil, fmt.Errorf("parse knowledge.index.yaml: %w", err)
	}
	out := make(map[string]bool, len(ki.Sources))
	for _, s := range ki.Sources {
		out[s.ID] = true
	}
	return out, nil
}

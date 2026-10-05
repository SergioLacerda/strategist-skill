package mission

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// loadOpportunityChestIndex is the mission composition adapter for the
// Opportunity Attack chest boundary. It reads the configured index as data;
// Treasure Chest mutation and domain validation remain outside the feat.
func loadOpportunityChestIndex(root string) (map[string]bool, error) {
	indexPath, err := opportunityIndexPath(root)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	raw, err := os.ReadFile(indexPath) //nolint:gosec // path is selected by the runtime configuration
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read knowledge index: %w", err)
	}
	return parseOpportunityChestIndex(raw)
}

func opportunityIndexPath(root string) (string, error) {
	activeRaw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // root is resolved by the CLI boundary
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Join(root, "knowledge.index.yaml"), nil
		}
		return "", fmt.Errorf("read active.yaml: %w", err)
	}
	var active struct {
		KnowledgeIndexPath string `yaml:"knowledge_index_path"`
	}
	if err := yaml.Unmarshal(activeRaw, &active); err != nil {
		return "", fmt.Errorf("parse active.yaml: %w", err)
	}
	indexPath := active.KnowledgeIndexPath
	if indexPath == "" {
		indexPath = "knowledge.index.yaml"
	}
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(root, indexPath)
	}
	return indexPath, nil
}

func parseOpportunityChestIndex(raw []byte) (map[string]bool, error) {
	var index struct {
		Sources []struct {
			ID string `yaml:"id"`
		} `yaml:"sources"`
	}
	if err := yaml.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("parse knowledge index: %w", err)
	}
	registered := make(map[string]bool, len(index.Sources))
	for _, source := range index.Sources {
		if source.ID != "" {
			registered[source.ID] = true
		}
	}
	return registered, nil
}

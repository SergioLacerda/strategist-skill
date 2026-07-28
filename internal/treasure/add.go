package treasure

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AddOptions holds the parameters for ExecuteAdd.
type AddOptions struct {
	ID         string
	Path       string
	Scope      string
	TrustTier  string
	ReviewedBy string
	Tags       []string
}

// ExecuteAdd registers a new treasure chest across active.yaml,
// treasure-chests.yaml, and knowledge.index.yaml in one pass. Mutations are
// applied via yaml.Node round-tripping so existing comments/formatting
// survive the edit.
func ExecuteAdd(root string, opts AddOptions) (indexPath string, err error) {
	activePath := filepath.Join(root, "active.yaml")
	governedPath := filepath.Join(root, "treasure-chests.yaml")
	indexPath = filepath.Join(root, "knowledge.index.yaml")

	activeDoc, governedDoc, indexDoc, err := LoadChestYAMLDocs(activePath, governedPath, indexPath)
	if err != nil {
		return "", fmt.Errorf("load chest YAML docs: %w", err)
	}
	if err := ApplyAddMutations(activeDoc, governedDoc, indexDoc, opts.ID, opts.Path, opts.Scope, opts.TrustTier, opts.ReviewedBy, opts.Tags); err != nil {
		return "", fmt.Errorf("apply add mutations: %w", err)
	}
	if err := writeTreasureChestAddDocs(activePath, governedPath, indexPath, activeDoc, governedDoc, indexDoc); err != nil {
		return "", err
	}
	return indexPath, nil
}

func writeTreasureChestAddDocs(activePath, governedPath, indexPath string, activeDoc, governedDoc, indexDoc *yaml.Node) error {
	written, err := WriteYAMLNodes(
		YAMLWrite{Path: activePath, Doc: activeDoc},
		YAMLWrite{Path: governedPath, Doc: governedDoc},
		YAMLWrite{Path: indexPath, Doc: indexDoc},
	)
	if err != nil {
		return fmt.Errorf("partial write after %v: %w", written, err)
	}
	return nil
}

package treasure

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ChestPaths holds the files mutated by treasure-chest add/remove operations.
type ChestPaths struct {
	Active   string
	Governed string
	Index    string
	Jewels   string
}

// NewChestPaths returns the standard treasure-chest file paths under a strategist root.
func NewChestPaths(root string) ChestPaths {
	return ChestPaths{
		Active:   filepath.Join(root, "active.yaml"),
		Governed: filepath.Join(root, "treasure-chests.yaml"),
		Index:    filepath.Join(root, "knowledge.index.yaml"),
		Jewels:   filepath.Join(root, "jewels.yaml"),
	}
}

// ChestDocSet holds loaded YAML documents for a chest mutation.
type ChestDocSet struct {
	Active   *yaml.Node
	Governed *yaml.Node
	Index    *yaml.Node
	Jewels   []YAMLWrite
}

// LoadChestYAMLDocs loads the three required YAML documents for adding a chest.
func LoadChestYAMLDocs(activePath, governedPath, indexPath string) (activeDoc, governedDoc, indexDoc *yaml.Node, err error) {
	activeDoc, err = ReadYAMLNode(activePath)
	if err != nil {
		return nil, nil, nil, err
	}
	governedDoc, err = ReadYAMLNode(governedPath)
	if err != nil {
		return nil, nil, nil, err
	}
	indexDoc, err = ReadYAMLNode(indexPath)
	if err != nil {
		return nil, nil, nil, err
	}
	return activeDoc, governedDoc, indexDoc, nil
}

// ApplyAddMutations appends a new chest to active, governed, and indexed layers.
func ApplyAddMutations(activeDoc, governedDoc, indexDoc *yaml.Node, id, path, scope, trustTier, reviewedBy string, tags []string) error {
	if err := AppendActiveChestEntry(activeDoc, id, path, scope); err != nil {
		return err
	}
	if err := AppendGovernedChestEntry(governedDoc, id, path, trustTier, reviewedBy, tags); err != nil {
		return err
	}
	if err := AppendIndexedSourceEntry(indexDoc, id, path, tags); err != nil {
		return err
	}
	return nil
}

// CheckChestIDAvailable errors if id is already registered and active.
func CheckChestIDAvailable(root, id string) error {
	activeChests, err := LoadActiveChests(root)
	if err != nil {
		return err
	}
	for _, ac := range activeChests {
		if ac.ID == id {
			return fmt.Errorf("id %q is already registered in active.yaml; use a different --id or remove it first", id)
		}
	}
	return nil
}

// DeriveChestIDFromPath derives the default chest id from the last path segment.
func DeriveChestIDFromPath(path string) string {
	path = strings.TrimRight(path, "/")
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// ParseTagsFlag parses a comma-separated task_type list. Blank input means all.
func ParseTagsFlag(tags string) []string {
	if strings.TrimSpace(tags) == "" {
		return []string{"all"}
	}
	parts := strings.Split(tags, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"all"}
	}
	return out
}

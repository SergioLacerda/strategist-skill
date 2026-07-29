package treasure

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveRemoveTarget resolves the chest id to remove from a positional path and/or
// --id flag, rejecting ambiguous input (both given but disagreeing).
func ResolveRemoveTarget(root, pathArg, idFlag string) (string, error) {
	if pathArg == "" {
		return idFlag, nil
	}

	matches, err := chestIDsForPath(root, pathArg)
	if err != nil {
		return "", err
	}
	return resolveRemoveMatches(pathArg, idFlag, matches)
}

func chestIDsForPath(root, pathArg string) ([]string, error) {
	activeChests, err := LoadActiveChests(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, ac := range activeChests {
		if ac.Path == pathArg {
			matches = append(matches, ac.ID)
		}
	}
	return matches, nil
}

func resolveRemoveMatches(pathArg, idFlag string, matches []string) (string, error) {
	switch len(matches) {
	case 0:
		if idFlag != "" {
			return idFlag, nil
		}
		return "", fmt.Errorf("no chest registered with path %q", pathArg)
	case 1:
		if idFlag != "" && idFlag != matches[0] {
			return "", fmt.Errorf("ambiguous — path %q resolves to id %q but --id=%q was given", pathArg, matches[0], idFlag)
		}
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous — multiple chests share path %q (%s); use --id", pathArg, strings.Join(matches, ", "))
	}
}

// LoadRemoveDocs loads the YAML documents required for removing a chest.
func LoadRemoveDocs(p ChestPaths) (ChestDocSet, error) {
	var docs ChestDocSet
	var err error
	docs.Active, err = ReadYAMLNode(p.Active)
	if err != nil {
		return docs, err
	}
	docs.Governed, err = ReadYAMLNode(p.Governed)
	if err != nil {
		return docs, err
	}
	docs.Index, err = ReadYAMLNode(p.Index)
	if err != nil {
		return docs, err
	}
	jewelPaths, err := JewelManifestPaths(filepath.Dir(p.Active))
	if err != nil {
		return docs, err
	}
	for _, path := range jewelPaths {
		jewelsDoc, jewelsErr := ReadYAMLNode(path)
		if jewelsErr != nil {
			return docs, jewelsErr
		}
		docs.Jewels = append(docs.Jewels, YAMLWrite{Path: path, Doc: jewelsDoc})
	}
	return docs, nil
}

// ApplyRemoveMutations removes/tombstones a chest across all loaded documents.
func ApplyRemoveMutations(docs ChestDocSet, id string) error {
	if err := RemoveActiveChestEntry(docs.Active, id); err != nil {
		return err
	}
	if err := MarkGovernedChestInactive(docs.Governed, id); err != nil {
		return err
	}
	if err := MarkIndexedSourceInactive(docs.Index, id); err != nil {
		return err
	}
	for _, jewels := range docs.Jewels {
		if err := MarkJewelsDeprecatedForChest(jewels.Doc, id); err != nil {
			return err
		}
	}
	return nil
}

// WriteRemoveDocs writes the documents changed by a remove operation.
func WriteRemoveDocs(p ChestPaths, docs ChestDocSet) ([]string, error) {
	writes := []YAMLWrite{
		{Path: p.Active, Doc: docs.Active},
		{Path: p.Governed, Doc: docs.Governed},
		{Path: p.Index, Doc: docs.Index},
	}
	writes = append(writes, docs.Jewels...)
	return WriteYAMLNodes(writes...)
}

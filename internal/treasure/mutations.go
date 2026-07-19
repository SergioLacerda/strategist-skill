package treasure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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

// PromoteJewel updates a jewel lifecycle status while preserving YAML structure.
func PromoteJewel(root, id, newStatus, evidenceRef string, reviewedAt time.Time) error {
	path, doc, entry, err := FindJewelDocument(root, id)
	if err != nil {
		return err
	}

	current := ""
	if v := MappingValue(entry, "status"); v != nil {
		current = v.Value
	}
	if current == domain.JewelStatusDeprecated && newStatus != domain.JewelStatusDeprecated {
		return fmt.Errorf("jewel %q is deprecated, cannot promote to %s", id, newStatus)
	}

	SetMappingField(entry, "status", newStatus)
	SetMappingField(entry, "reviewed_by", "human")
	reviewedDate := reviewedAt.UTC().Format("2006-01-02")
	SetMappingField(entry, "last_reviewed", reviewedDate)
	if newStatus == domain.JewelStatusVerified {
		AppendEvidenceRef(entry, evidenceRef)
	}
	AppendJewelHistory(entry, newStatus, reviewedDate, "human", evidenceRef)

	if _, err := WriteYAMLNodes(YAMLWrite{Path: path, Doc: doc}); err != nil {
		return err
	}
	return nil
}

// MigrateLegacyJewelStatus rewrites legacy status: active jewels to status: accepted.
func MigrateLegacyJewelStatus(root string) (int, error) {
	writes, migrated, err := legacyJewelStatusWrites(root)
	if err != nil || migrated == 0 {
		return migrated, err
	}
	if _, err := WriteYAMLNodes(writes...); err != nil {
		return migrated, err
	}
	return migrated, nil
}

func legacyJewelStatusWrites(root string) ([]YAMLWrite, int, error) {
	paths, err := JewelManifestPaths(root)
	if err != nil {
		return nil, 0, err
	}
	migrated := 0
	var writes []YAMLWrite
	for _, path := range paths {
		write, count, err := legacyJewelStatusWrite(path)
		if err != nil {
			return nil, migrated, err
		}
		migrated += count
		if count > 0 {
			writes = append(writes, write)
		}
	}
	return writes, migrated, nil
}

func legacyJewelStatusWrite(path string) (YAMLWrite, int, error) {
	doc, err := ReadYAMLNode(path)
	if err != nil {
		return YAMLWrite{}, 0, err
	}
	count, err := MigrateLegacyJewelStatusInDocument(doc)
	if err != nil {
		return YAMLWrite{}, 0, err
	}
	return YAMLWrite{Path: path, Doc: doc}, count, nil
}

// MigrateLegacyJewelStatusInDocument rewrites legacy status: active entries in one manifest.
func MigrateLegacyJewelStatusInDocument(doc *yaml.Node) (int, error) {
	mapping, err := RootMapping(doc)
	if err != nil {
		return 0, err
	}
	seq := MappingValue(mapping, "jewels")
	if seq == nil {
		return 0, nil
	}
	migrated := 0
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		v := MappingValue(entry, "status")
		if v != nil && v.Value == "active" {
			SetMappingField(entry, "status", domain.JewelStatusAccepted)
			migrated++
		}
	}
	return migrated, nil
}

// FindJewelDocument returns the manifest document and jewel mapping for id across
// monolithic and partitioned jewel manifests.
func FindJewelDocument(root, id string) (string, *yaml.Node, *yaml.Node, error) {
	paths, err := JewelManifestPaths(root)
	if err != nil {
		return "", nil, nil, err
	}
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return "", nil, nil, err
		}
		entry, err := FindJewelEntry(doc, id)
		if err == nil {
			return path, doc, entry, nil
		}
	}
	return "", nil, nil, fmt.Errorf("jewel %q not found in jewels.yaml or jewels/", id)
}

// NewJewelsDocument creates an empty schema-versioned jewels manifest.
func NewJewelsDocument() *yaml.Node {
	root := mapNode(
		strNode("schema_version"), strNode("1"),
		strNode("jewels"), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
	)
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
}

// ReadOrCreateJewelsDocument loads a jewels manifest or creates an empty one.
func ReadOrCreateJewelsDocument(path string) (*yaml.Node, error) {
	doc, err := ReadYAMLNode(path)
	if err == nil {
		return doc, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return NewJewelsDocument(), nil
	}
	return nil, err
}

// JewelPartitionPath returns the partition manifest path for a chest id.
func JewelPartitionPath(root, chestID string) string {
	name := strings.NewReplacer("/", "_", "\\", "_").Replace(chestID)
	return filepath.Join(root, "jewels", name+".yaml")
}

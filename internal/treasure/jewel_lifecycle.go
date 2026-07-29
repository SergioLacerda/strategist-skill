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

// ErrJewelNotFound is returned by FindJewelDocument (and callers like PromoteJewel) when
// no jewel with the given id exists — lets callers that also check Potion (e.g. the
// `items` CLI dispatch) distinguish "not a jewel" from a genuine I/O or validation error.
var ErrJewelNotFound = errors.New("jewel not found")

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
		if migrateLegacyJewelStatus(entry) {
			SetMappingField(entry, "status", domain.JewelStatusAccepted)
			migrated++
		}
	}
	return migrated, nil
}

func migrateLegacyJewelStatus(entry *yaml.Node) bool {
	if entry.Kind != yaml.MappingNode {
		return false
	}
	v := MappingValue(entry, "status")
	return v != nil && v.Value == "active"
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
	return "", nil, nil, fmt.Errorf("jewel %q: %w (checked jewels.yaml and jewels/)", id, ErrJewelNotFound)
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

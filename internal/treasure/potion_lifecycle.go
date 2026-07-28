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

// --- lifecycle ---

// ErrPotionNotFound is returned when no potion with the given id exists — lets callers
// that also check Jewel (the `items` CLI dispatch) distinguish "not a potion" from a
// genuine I/O or validation error.
var ErrPotionNotFound = errors.New("potion not found")

// PromotePotion updates a potion lifecycle status while preserving YAML structure.
// Mirrors PromoteJewel. evidenceRef is accepted for CLI-surface parity with
// PromoteJewel/`items verify --evidence` but not persisted: the Potion schema (see
// internal/embed/defaults/potions.yaml) has no verification/evidence_refs field, and
// this function must not redecide that schema.
func PromotePotion(root, id, newStatus, _ string, reviewedAt time.Time) error {
	path, doc, entry, err := FindPotionDocument(root, id)
	if err != nil {
		return err
	}

	current := ""
	if v := MappingValue(entry, "status"); v != nil {
		current = v.Value
	}
	if current == domain.PotionStatusDeprecated && newStatus != domain.PotionStatusDeprecated {
		return fmt.Errorf("potion %q is deprecated, cannot promote to %s", id, newStatus)
	}

	SetMappingField(entry, "status", newStatus)
	SetMappingField(entry, "reviewed_by", "human")
	SetMappingField(entry, "last_reviewed", reviewedAt.UTC().Format("2006-01-02"))

	if _, err := WriteYAMLNodes(YAMLWrite{Path: path, Doc: doc}); err != nil {
		return err
	}
	return nil
}

// FindPotionDocument returns the manifest document and potion mapping for id across
// monolithic and partitioned potion manifests.
func FindPotionDocument(root, id string) (string, *yaml.Node, *yaml.Node, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return "", nil, nil, err
	}
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return "", nil, nil, err
		}
		entry, err := FindPotionEntry(doc, id)
		if err == nil {
			return path, doc, entry, nil
		}
	}
	return "", nil, nil, fmt.Errorf("potion %q: %w (checked potions.yaml and potions/)", id, ErrPotionNotFound)
}

// FindPotionEntry returns the potions.yaml potion entry mapping node for id, or an
// error if no potions are declared or id is not found.
func FindPotionEntry(doc *yaml.Node, id string) (*yaml.Node, error) {
	root, err := RootMapping(doc)
	if err != nil {
		return nil, err
	}
	seq := MappingValue(root, "potions")
	if seq == nil {
		return nil, fmt.Errorf("no potions declared in potions.yaml")
	}
	entry, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return nil, fmt.Errorf("potion %q not found in potions.yaml", id)
	}
	return entry, nil
}

// NewPotionsDocument creates an empty schema-versioned potions manifest.
func NewPotionsDocument() *yaml.Node {
	root := mapNode(
		strNode("schema_version"), strNode("1"),
		strNode("potions"), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
	)
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
}

// ReadOrCreatePotionsDocument loads a potions manifest or creates an empty one.
func ReadOrCreatePotionsDocument(path string) (*yaml.Node, error) {
	doc, err := ReadYAMLNode(path)
	if err == nil {
		return doc, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return NewPotionsDocument(), nil
	}
	return nil, err
}

// PotionPartitionPath returns the partition manifest path for a chest id.
func PotionPartitionPath(root, chestID string) string {
	name := strings.NewReplacer("/", "_", "\\", "_").Replace(chestID)
	return filepath.Join(root, "potions", name+".yaml")
}

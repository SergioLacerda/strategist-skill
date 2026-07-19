package treasure

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// yaml.Node-based read-modify-write helpers for treasure-chest add/remove (SQ-006 / T-I).
// Every other YAML writer in this codebase either struct-marshals (losing comments) or
// does an install-time-only placeholder replace; these helpers preserve existing comments
// and formatting when mutating an already-populated file.

// YAMLWrite pairs a destination path with the YAML document to write there.
type YAMLWrite struct {
	Path string
	Doc  *yaml.Node
}

// ReadYAMLNode reads a YAML file into a document node.
func ReadYAMLNode(path string) (*yaml.Node, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
}

// WriteYAMLNodes writes each document to its file, in order, stopping at the first
// failure. It returns the list of paths successfully written so callers can report
// exactly how much state was mutated if a later write fails.
func WriteYAMLNodes(writes ...YAMLWrite) ([]string, error) {
	var written []string
	for _, w := range writes {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2) // matches this repo's existing YAML style
		if err := enc.Encode(w.Doc); err != nil {
			return written, fmt.Errorf("encode %s: %w", w.Path, err)
		}
		if err := enc.Close(); err != nil {
			return written, fmt.Errorf("encode %s: %w", w.Path, err)
		}
		if err := os.WriteFile(w.Path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // G306
			return written, fmt.Errorf("write %s: %w", w.Path, err)
		}
		written = append(written, w.Path)
	}
	return written, nil
}

// RootMapping returns the top-level mapping node from a YAML document.
func RootMapping(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("yaml document has no content")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("yaml root is not a mapping")
	}
	return root, nil
}

// MappingValue returns the value node for a mapping key, or nil when absent.
func MappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// FindOrCreateSequence returns the sequence node for key under mapping, creating an
// empty one (and appending the key) if it does not exist yet.
func FindOrCreateSequence(mapping *yaml.Node, key string) *yaml.Node {
	if v := MappingValue(mapping, key); v != nil {
		return v
	}
	keyNode := strNode(key)
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	mapping.Content = append(mapping.Content, keyNode, seq)
	return seq
}

func strNode(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

func flowSeqNode(values []string) *yaml.Node {
	items := make([]*yaml.Node, 0, len(values))
	for _, v := range values {
		items = append(items, strNode(v))
	}
	return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle, Content: items}
}

func mapNode(pairs ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: pairs}
}

// FindEntryByID returns the mapping node and index within seq.Content whose "id"
// field equals id, or nil/-1 if not found.
func FindEntryByID(seq *yaml.Node, id string) (*yaml.Node, int) {
	for i, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if v := MappingValue(item, "id"); v != nil && v.Value == id {
			return item, i
		}
	}
	return nil, -1
}

// SetMappingField sets key's value to a string scalar on mapping, adding the pair if
// key does not already exist.
func SetMappingField(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = strNode(value)
			return
		}
	}
	mapping.Content = append(mapping.Content, strNode(key), strNode(value))
}

// AppendActiveChestEntry appends an active.yaml treasure_chests entry.
func AppendActiveChestEntry(doc *yaml.Node, id, path, scope string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := FindOrCreateSequence(root, "treasure_chests")
	entry := mapNode(
		strNode("id"), strNode(id),
		strNode("path"), strNode(path),
		strNode("scope"), strNode(scope),
	)
	seq.Content = append(seq.Content, entry)
	return nil
}

// RemoveActiveChestEntry removes an active.yaml treasure_chests entry by id.
func RemoveActiveChestEntry(doc *yaml.Node, id string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(root, "treasure_chests")
	if seq == nil {
		return fmt.Errorf("no treasure_chests declared in active.yaml")
	}
	_, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in active.yaml treasure_chests", id)
	}
	seq.Content = append(seq.Content[:idx], seq.Content[idx+1:]...)
	return nil
}

// AppendGovernedChestEntry appends a treasure-chests.yaml governed chest entry.
func AppendGovernedChestEntry(doc *yaml.Node, id, path, trustTier, reviewedBy string, tags []string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := FindOrCreateSequence(root, "chests")
	entry := mapNode(
		strNode("id"), strNode(id),
		strNode("title"), strNode(id),
		strNode("path"), strNode(path),
		strNode("trust"), mapNode(
			strNode("tier"), strNode(trustTier),
			strNode("reviewed_by"), strNode(reviewedBy),
		),
		strNode("routing"), mapNode(
			strNode("task_types"), flowSeqNode(tags),
		),
		strNode("retrieval"), mapNode(
			strNode("strategy"), strNode("selective"),
			strNode("require_relevance_reason"), boolNode(true),
			strNode("allow_full_load"), boolNode(false),
		),
	)
	seq.Content = append(seq.Content, entry)
	return nil
}

// MarkGovernedChestInactive sets status: inactive for a governed chest entry.
func MarkGovernedChestInactive(doc *yaml.Node, id string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(root, "chests")
	if seq == nil {
		return fmt.Errorf("no chests declared in treasure-chests.yaml")
	}
	entry, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in treasure-chests.yaml chests", id)
	}
	SetMappingField(entry, "status", "inactive")
	return nil
}

// AppendIndexedSourceEntry appends a knowledge.index.yaml source entry.
func AppendIndexedSourceEntry(doc *yaml.Node, id, path string, tags []string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := FindOrCreateSequence(root, "sources")
	entry := mapNode(
		strNode("id"), strNode(id),
		strNode("path"), strNode(path),
		strNode("tags"), flowSeqNode(tags),
	)
	seq.Content = append(seq.Content, entry)
	return nil
}

// MarkIndexedSourceInactive sets status: inactive for an indexed source entry.
func MarkIndexedSourceInactive(doc *yaml.Node, id string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(root, "sources")
	if seq == nil {
		return fmt.Errorf("no sources declared in knowledge.index.yaml")
	}
	entry, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in knowledge.index.yaml sources", id)
	}
	SetMappingField(entry, "status", "inactive")
	return nil
}

// --- jewels.yaml: jewels[] ---

// MarkJewelsDeprecatedForChest sets status: deprecated on every jewels.yaml entry whose
// chest_id matches chestID — the chest-removal cascade (SQ-009). A missing jewels: sequence
// or zero matches is not an error: a chest may have no jewels yet.
func MarkJewelsDeprecatedForChest(doc *yaml.Node, chestID string) error {
	root, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(root, "jewels")
	if seq == nil {
		return nil
	}
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		if v := MappingValue(entry, "chest_id"); v != nil && v.Value == chestID {
			SetMappingField(entry, "status", "deprecated")
			AppendJewelHistory(entry, "deprecated", time.Now().UTC().Format("2006-01-02"), "human", "")
		}
	}
	return nil
}

// FindJewelEntry returns the jewels.yaml jewel entry mapping node for id, or an error if
// no jewels are declared or id is not found.
func FindJewelEntry(doc *yaml.Node, id string) (*yaml.Node, error) {
	root, err := RootMapping(doc)
	if err != nil {
		return nil, err
	}
	seq := MappingValue(root, "jewels")
	if seq == nil {
		return nil, fmt.Errorf("no jewels declared in jewels.yaml")
	}
	entry, idx := FindEntryByID(seq, id)
	if idx == -1 {
		return nil, fmt.Errorf("jewel %q not found in jewels.yaml", id)
	}
	return entry, nil
}

// AppendEvidenceRef appends ref to entry's verification.evidence_refs sequence, creating
// the verification mapping if it does not already exist (e.g. on a jewel hand-authored
// before the verification field was introduced).
func AppendEvidenceRef(entry *yaml.Node, ref string) {
	verification := MappingValue(entry, "verification")
	if verification == nil {
		verification = mapNode()
		entry.Content = append(entry.Content, strNode("verification"), verification)
	}
	seq := FindOrCreateSequence(verification, "evidence_refs")
	seq.Content = append(seq.Content, strNode(ref))
}

// AppendJewelHistory appends a lifecycle history entry to a jewel mapping.
func AppendJewelHistory(entry *yaml.Node, status, at, by, evidenceRef string) {
	seq := FindOrCreateSequence(entry, "history")
	historyEntry := mapNode(
		strNode("status"), strNode(status),
		strNode("at"), strNode(at),
		strNode("by"), strNode(by),
	)
	if evidenceRef != "" {
		historyEntry.Content = append(historyEntry.Content, strNode("evidence_ref"), strNode(evidenceRef))
	}
	seq.Content = append(seq.Content, historyEntry)
}

func boolNode(v bool) *yaml.Node {
	val := "false"
	if v {
		val = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: val}
}

package main

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// yaml.Node-based read-modify-write helpers for treasure-chest add/remove (SQ-006 / T-I).
// Every other YAML writer in this codebase either struct-marshals (losing comments) or
// does an install-time-only placeholder replace; these helpers preserve existing comments
// and formatting when mutating an already-populated file.

type yamlWrite struct {
	path string
	doc  *yaml.Node
}

func readYAMLNode(path string) (*yaml.Node, error) {
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

// writeYAMLNodes writes each document to its file, in order, stopping at the first
// failure. It returns the list of paths successfully written so callers can report
// exactly how much state was mutated if a later write fails.
func writeYAMLNodes(writes ...yamlWrite) ([]string, error) {
	var written []string
	for _, w := range writes {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2) // matches this repo's existing YAML style
		if err := enc.Encode(w.doc); err != nil {
			return written, fmt.Errorf("encode %s: %w", w.path, err)
		}
		if err := enc.Close(); err != nil {
			return written, fmt.Errorf("encode %s: %w", w.path, err)
		}
		if err := os.WriteFile(w.path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // G306
			return written, fmt.Errorf("write %s: %w", w.path, err)
		}
		written = append(written, w.path)
	}
	return written, nil
}

func rootMapping(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("yaml document has no content")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("yaml root is not a mapping")
	}
	return root, nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// findOrCreateSequence returns the sequence node for key under mapping, creating an
// empty one (and appending the key) if it does not exist yet.
func findOrCreateSequence(mapping *yaml.Node, key string) *yaml.Node {
	if v := mappingValue(mapping, key); v != nil {
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

// findEntryByID returns the mapping node and index within seq.Content whose "id"
// field equals id, or nil/-1 if not found.
func findEntryByID(seq *yaml.Node, id string) (*yaml.Node, int) {
	for i, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if v := mappingValue(item, "id"); v != nil && v.Value == id {
			return item, i
		}
	}
	return nil, -1
}

// setMappingField sets key's value to a string scalar on mapping, adding the pair if
// key does not already exist.
func setMappingField(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = strNode(value)
			return
		}
	}
	mapping.Content = append(mapping.Content, strNode(key), strNode(value))
}

// --- active.yaml: treasure_chests[] ---

func appendActiveChestEntry(doc *yaml.Node, id, path, scope string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := findOrCreateSequence(root, "treasure_chests")
	entry := mapNode(
		strNode("id"), strNode(id),
		strNode("path"), strNode(path),
		strNode("scope"), strNode(scope),
	)
	seq.Content = append(seq.Content, entry)
	return nil
}

func removeActiveChestEntry(doc *yaml.Node, id string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := mappingValue(root, "treasure_chests")
	if seq == nil {
		return fmt.Errorf("no treasure_chests declared in active.yaml")
	}
	_, idx := findEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in active.yaml treasure_chests", id)
	}
	seq.Content = append(seq.Content[:idx], seq.Content[idx+1:]...)
	return nil
}

// --- treasure-chests.yaml: chests[] ---

func appendGovernedChestEntry(doc *yaml.Node, id, path, trustTier, reviewedBy string, tags []string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := findOrCreateSequence(root, "chests")
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

func markGovernedChestInactive(doc *yaml.Node, id string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := mappingValue(root, "chests")
	if seq == nil {
		return fmt.Errorf("no chests declared in treasure-chests.yaml")
	}
	entry, idx := findEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in treasure-chests.yaml chests", id)
	}
	setMappingField(entry, "status", "inactive")
	return nil
}

// --- knowledge.index.yaml: sources[] ---

func appendIndexedSourceEntry(doc *yaml.Node, id, path string, tags []string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := findOrCreateSequence(root, "sources")
	entry := mapNode(
		strNode("id"), strNode(id),
		strNode("path"), strNode(path),
		strNode("tags"), flowSeqNode(tags),
	)
	seq.Content = append(seq.Content, entry)
	return nil
}

func markIndexedSourceInactive(doc *yaml.Node, id string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := mappingValue(root, "sources")
	if seq == nil {
		return fmt.Errorf("no sources declared in knowledge.index.yaml")
	}
	entry, idx := findEntryByID(seq, id)
	if idx == -1 {
		return fmt.Errorf("id %q not found in knowledge.index.yaml sources", id)
	}
	setMappingField(entry, "status", "inactive")
	return nil
}

// --- jewels.yaml: jewels[] ---

// markJewelsDeprecatedForChest sets status: deprecated on every jewels.yaml entry whose
// chest_id matches chestID — the chest-removal cascade (SQ-009). A missing jewels: sequence
// or zero matches is not an error: a chest may have no jewels yet.
func markJewelsDeprecatedForChest(doc *yaml.Node, chestID string) error {
	root, err := rootMapping(doc)
	if err != nil {
		return err
	}
	seq := mappingValue(root, "jewels")
	if seq == nil {
		return nil
	}
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		if v := mappingValue(entry, "chest_id"); v != nil && v.Value == chestID {
			setMappingField(entry, "status", "deprecated")
		}
	}
	return nil
}

func boolNode(v bool) *yaml.Node {
	val := "false"
	if v {
		val = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: val}
}

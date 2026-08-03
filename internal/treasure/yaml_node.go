package treasure

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// yaml.Node-based read-modify-write helpers for treasure-chest add/remove (SQ-006 / T-I).
// Every other YAML writer in this codebase either struct-marshals (losing comments) or
// does an install-time-only placeholder replace; these helpers preserve existing comments
// and formatting when mutating an already-populated file.

// ReadYAMLNode reads a YAML file into a document node.
func ReadYAMLNode(path string) (*yaml.Node, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: YAML mutation reads files selected by governed treasure operations
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
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

func boolNode(v bool) *yaml.Node {
	val := "false"
	if v {
		val = "true"
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: val}
}

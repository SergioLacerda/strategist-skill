package treasure

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// --- index write (ask #1 / SQ-001) ---

// WriteProposedPotions appends new proposed potions while skipping existing ids.
// Mirrors WriteProposedJewels.
func WriteProposedPotions(root string, candidates []Potion) (written, skipped int, err error) {
	if len(candidates) == 0 {
		return 0, 0, nil
	}

	docs, written, skipped, err := buildProposedPotionDocs(root, candidates)
	if err != nil {
		return written, skipped, err
	}

	if written == 0 {
		return written, skipped, nil
	}
	if err := writeProposedPotionDocs(root, docs); err != nil {
		return written, skipped, err
	}
	return written, skipped, nil
}

func buildProposedPotionDocs(root string, candidates []Potion) (map[string]*yaml.Node, int, int, error) {
	existing, err := ExistingPotionIDs(root)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("index proposed potions: %w", err)
	}
	docs := make(map[string]*yaml.Node)
	written, skipped := 0, 0
	for _, cand := range candidates {
		if existing[cand.ID] {
			skipped++
			continue
		}
		if err := appendCandidateToPotionPartition(root, docs, cand); err != nil {
			return docs, written, skipped, err
		}
		existing[cand.ID] = true
		written++
	}
	return docs, written, skipped, nil
}

func writeProposedPotionDocs(root string, docs map[string]*yaml.Node) error {
	return writeProposedItemDocs(root, "potions", docs, "index proposed potions")
}

func appendCandidateToPotionPartition(root string, docs map[string]*yaml.Node, cand Potion) error {
	path := PotionPartitionPath(root, cand.ChestID)
	doc, ok := docs[path]
	if !ok {
		var err error
		doc, err = ReadOrCreatePotionsDocument(path)
		if err != nil {
			return fmt.Errorf("index proposed potions: %w", err)
		}
		docs[path] = doc
	}
	mapping, err := RootMapping(doc)
	if err != nil {
		return fmt.Errorf("index proposed potions: %w", err)
	}
	seq := FindOrCreateSequence(mapping, "potions")
	seq.Style = 0
	var entry yaml.Node
	if err := entry.Encode(cand); err != nil {
		return fmt.Errorf("index proposed potions: encode %s: %w", cand.ID, err)
	}
	seq.Content = append(seq.Content, &entry)
	return nil
}

// ExistingPotionIDs returns all ids present across monolithic and partitioned potion manifests.
func ExistingPotionIDs(root string) (map[string]bool, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool)
	for _, path := range paths {
		doc, err := ReadYAMLNode(path)
		if err != nil {
			return nil, err
		}
		if err := collectPotionIDs(doc, ids); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func collectPotionIDs(doc *yaml.Node, ids map[string]bool) error {
	mapping, err := RootMapping(doc)
	if err != nil {
		return err
	}
	seq := MappingValue(mapping, "potions")
	if seq == nil {
		return nil
	}
	for _, entry := range seq.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		if id := MappingValue(entry, "id"); id != nil {
			ids[id.Value] = true
		}
	}
	return nil
}

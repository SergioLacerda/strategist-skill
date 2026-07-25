package treasure

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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

package treasure

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

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
		deprecateJewelForChest(entry, chestID)
	}
	return nil
}

func deprecateJewelForChest(entry *yaml.Node, chestID string) {
	if entry.Kind != yaml.MappingNode {
		return
	}
	if v := MappingValue(entry, "chest_id"); v != nil && v.Value == chestID {
		SetMappingField(entry, "status", "deprecated")
		AppendJewelHistory(entry, "deprecated", time.Now().UTC().Format("2006-01-02"), "human", "")
	}
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

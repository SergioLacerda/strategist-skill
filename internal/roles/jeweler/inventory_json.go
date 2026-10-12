package jeweler

import (
	"encoding/json"
	"fmt"
	"sort"
)

// MarshalInventory returns deterministic JSON for callers that persist the
// read-only capture before building a migration manifest.
func MarshalInventory(entries []InventoryEntry) ([]byte, error) {
	canonical := append([]InventoryEntry(nil), entries...)
	if err := validateInventoryEntries(canonical); err != nil {
		return nil, err
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Key < canonical[j].Key })
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("jeweler: marshal inventory: %w", err)
	}
	return raw, nil
}

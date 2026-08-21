package install

import (
	"fmt"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

func inventoryFromLock(lock domain.PluginLock) []domain.InstalledInstance {
	instances := make([]domain.InstalledInstance, 0, len(lock.Nodes))
	for _, node := range lock.Nodes {
		instances = append(instances, domain.InstalledInstance{
			ID:          installedInstanceID(node),
			ConnectorID: "current-runtime",
			LockDigest:  lock.GraphDigest,
			State:       lifecycle.StateResolved,
		})
		if node.Kind == "adapter_contract" {
			instances[len(instances)-1].AdapterDigest = node.Digest
		}
		if node.Kind == "package" {
			instances[len(instances)-1].PackageDigest = node.Digest
		}
	}
	return instances
}

func bindingsFromSlots(slots map[string]string, lock domain.PluginLock) ([]domain.SlotBinding, error) {
	nodesByID := map[string]domain.PluginLockNode{}
	for _, node := range lock.Nodes {
		nodesByID[node.ID] = node
	}
	bindings := make([]domain.SlotBinding, 0, len(slots))
	for _, slot := range sortedSlotNames(slots) {
		node, ok := nodesByID[slots[slot]]
		if !ok {
			return nil, fmt.Errorf("unresolved_binding: %s provider %s", slot, slots[slot])
		}
		bindings = append(bindings, domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			Slot:                slot,
			InstalledInstanceID: installedInstanceID(node),
			Generation:          0,
			Status:              "enabled",
			NativeFallback:      nativeFallbackForSlot(slot),
		})
	}
	return bindings, nil
}

func mergeInventory(store *lifecycle.Store, instances []domain.InstalledInstance) {
	seen := map[string]bool{}
	for _, instance := range store.Inventory.Instances {
		seen[instance.ID] = true
	}
	for _, instance := range instances {
		if !seen[instance.ID] {
			store.Inventory.Instances = append(store.Inventory.Instances, instance)
		}
	}
}

func changesFromBindings(bindings []domain.SlotBinding) []string {
	changes := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		changes = append(changes, fmt.Sprintf("slot %s -> %s", binding.Slot, binding.InstalledInstanceID))
	}
	return changes
}

func installedInstanceID(node domain.PluginLockNode) string {
	return node.ID + "@" + node.Digest
}

func sortedSlotNames(slots map[string]string) []string {
	names := make([]string, 0, len(slots))
	for slot := range slots {
		names = append(names, slot)
	}
	sort.Strings(names)
	return names
}

func nativeFallbackForSlot(slot string) string {
	switch slot {
	case "discovery":
		return "ranger"
	case "refinement":
		return "archivist"
	case "execution":
		return "sniper"
	default:
		return ""
	}
}

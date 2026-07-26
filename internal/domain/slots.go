package domain

// SlotName identifies one Strategist pipeline slot.
type SlotName string

const (
	// SlotDiscovery is the canonical discovery pipeline slot.
	SlotDiscovery SlotName = "discovery"

	// SlotRefinement is the canonical refinement pipeline slot.
	SlotRefinement SlotName = "refinement"

	// SlotExecution is the canonical execution pipeline slot.
	SlotExecution SlotName = "execution"
)

const requiredSlotList = "discovery, refinement, execution"

var validSlots = stringSet(
	string(SlotDiscovery),
	string(SlotRefinement),
	string(SlotExecution),
)

// RequiredSlots returns the ordered slot vocabulary required by runtime config.
func RequiredSlots() []SlotName {
	return []SlotName{SlotDiscovery, SlotRefinement, SlotExecution}
}

// IsValidSlot reports whether slot is a known Strategist pipeline slot.
func IsValidSlot(slot string) bool {
	return hasString(validSlots, slot)
}

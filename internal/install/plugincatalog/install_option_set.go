package plugincatalog

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// InstallOptionSet is the catalog-derived option surface shared by Wizard and
// headless planning. It is read-only; selection and activation remain separate
// operations owned by their callers.
type InstallOptionSet struct {
	bySlot map[string]SlotOptions
}

// NewInstallOptionSet derives the per-slot option surface from catalog.
func NewInstallOptionSet(catalog PluginCatalog) InstallOptionSet {
	return InstallOptionSet{bySlot: map[string]SlotOptions{
		string(domain.SlotDiscovery):  CompatibleSlotOptions(catalog, SlotRoleID(domain.SlotDiscovery), SlotHandoffSchema(domain.SlotDiscovery)),
		string(domain.SlotRefinement): CompatibleSlotOptions(catalog, SlotRoleID(domain.SlotRefinement), SlotHandoffSchema(domain.SlotRefinement)),
		string(domain.SlotExecution):  CompatibleSlotOptions(catalog, SlotRoleID(domain.SlotExecution), SlotHandoffSchema(domain.SlotExecution)),
	}}
}

// ForSlot returns the options available for slot.
func (s InstallOptionSet) ForSlot(slot string) SlotOptions {
	return s.bySlot[slot]
}

// ValidateRankedSelections proves that a headless Ranked choice came from the
// same certified option set the Wizard presents. Custom selections remain
// extensible and are validated by catalog resolution and role affinity.
func (s InstallOptionSet) ValidateRankedSelections(slots, modes map[string]string) error {
	for slot, mode := range modes {
		if mode != domain.SlotBindingModeRanked {
			continue
		}
		provider := slots[slot]
		options := s.ForSlot(slot)
		if !containsOptionRef(options.RankedRefs, provider) {
			return fmt.Errorf("install options: Ranked selection %q is not certified for slot %q", provider, slot)
		}
	}
	return nil
}

func containsOptionRef(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

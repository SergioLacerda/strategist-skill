package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// installOptionSet is the catalog-derived option surface shared by Wizard and
// headless planning. It is read-only; selection and activation remain separate
// operations owned by their callers.
type installOptionSet struct {
	bySlot map[string]slotOptions
}

func newInstallOptionSet(catalog pluginCatalog) installOptionSet {
	return installOptionSet{bySlot: map[string]slotOptions{
		string(domain.SlotDiscovery):  compatibleSlotOptions(catalog, slotRoleID(domain.SlotDiscovery), slotHandoffSchema(domain.SlotDiscovery)),
		string(domain.SlotRefinement): compatibleSlotOptions(catalog, slotRoleID(domain.SlotRefinement), slotHandoffSchema(domain.SlotRefinement)),
		string(domain.SlotExecution):  compatibleSlotOptions(catalog, slotRoleID(domain.SlotExecution), slotHandoffSchema(domain.SlotExecution)),
	}}
}

func (s installOptionSet) forSlot(slot string) slotOptions {
	return s.bySlot[slot]
}

// ValidateRankedSelections proves that a headless Ranked choice came from the
// same certified option set the Wizard presents. Custom selections remain
// extensible and are validated by catalog resolution and role affinity.
func (s installOptionSet) ValidateRankedSelections(slots, modes map[string]string) error {
	for slot, mode := range modes {
		if mode != domain.SlotBindingModeRanked {
			continue
		}
		provider := slots[slot]
		options := s.forSlot(slot)
		if !containsOptionRef(options.rankedRefs, provider) {
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

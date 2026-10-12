package plugincatalog

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// WizardSlotModes returns the binding mode chosen for each slot in wc, keyed by slot name.
func WizardSlotModes(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		string(domain.SlotDiscovery):  wc.DiscoveryMode,
		string(domain.SlotRefinement): wc.RefinementMode,
		string(domain.SlotExecution):  wc.ExecutionMode,
	}
}

// ReplaceSlotBinding overwrites bindings' entry for replacement.Slot in
// place, or appends replacement if none exists yet.
func ReplaceSlotBinding(bindings []domain.SlotBinding, replacement domain.SlotBinding) []domain.SlotBinding {
	for i, b := range bindings {
		if b.Slot == replacement.Slot {
			bindings[i] = replacement
			return bindings
		}
	}
	return append(bindings, replacement)
}

// WizardSlots returns the provider chosen for each slot in wc, keyed by slot name.
func WizardSlots(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		string(domain.SlotDiscovery):  wc.DiscoveryProvider,
		string(domain.SlotRefinement): wc.RefinementProvider,
		string(domain.SlotExecution):  wc.ExecutionProvider,
	}
}

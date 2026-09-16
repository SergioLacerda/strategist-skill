package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// rankedModeSlots maps each Wizard-collected mode field to its slot name, so
// applyRankedBindingChoices can iterate without duplicating "discovery"/
// "refinement" across both fields.
func rankedModeSlots(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		"discovery":  wc.DiscoveryMode,
		"refinement": wc.RefinementMode,
	}
}

// applyRankedBindingChoices overwrites, for every slot the Wizard resolved
// to Ranked, the Custom-mode SlotBinding activateRoleProviderMigration just
// wrote with the pre-generated Ranked record from the catalog's
// certification stamp (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
// DEC-005) — an activate/copy step, not fresh construction. Every slot the
// Wizard resolved to Custom (the overwhelming majority: every existing
// installation, since DiscoveryMode/RefinementMode default to "") is
// returned untouched.
func applyRankedBindingChoices(catalog pluginCatalog, wc domain.WizardConfig, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	slots := wizardSlots(wc)
	for slot, mode := range rankedModeSlots(wc) {
		if mode != domain.SlotBindingModeRanked {
			continue
		}
		contract, err := rankedContractForProvider(catalog, slots[slot])
		if err != nil {
			return domain.PluginLockFile{}, fmt.Errorf("apply ranked binding: slot %s: %w", slot, err)
		}
		lockFile.Bindings = replaceSlotBinding(lockFile.Bindings, domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			Slot:                slot,
			InstalledInstanceID: contract.ID,
			Generation:          contract.RankedBindingGeneration,
			Status:              contract.RankedBindingStatus,
			Mode:                domain.SlotBindingModeRanked,
		})
	}
	return lockFile, nil
}

func rankedContractForProvider(catalog pluginCatalog, providerID string) (domain.ProviderContract, error) {
	provider, ok := findCatalogProvider(catalog, providerID)
	if !ok {
		return domain.ProviderContract{}, fmt.Errorf("provider %q not found in catalog", providerID)
	}
	contract := providerContractFromCatalogEntry(provider)
	if !contract.Ranked || contract.CertificationDigest == "" {
		return domain.ProviderContract{}, fmt.Errorf("provider %q is not a certified ranked candidate", providerID)
	}
	return contract, nil
}

// replaceSlotBinding overwrites bindings' entry for replacement.Slot in
// place, or appends replacement if none exists yet.
func replaceSlotBinding(bindings []domain.SlotBinding, replacement domain.SlotBinding) []domain.SlotBinding {
	for i, b := range bindings {
		if b.Slot == replacement.Slot {
			bindings[i] = replacement
			return bindings
		}
	}
	return append(bindings, replacement)
}

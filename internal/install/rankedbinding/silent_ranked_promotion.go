package rankedbinding

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
)

// PromoteRuntimeProvidersToRanked gives a silent install the same outcome the
// wizard's pre-selected Ranked option gives: every certified provider that
// declares a runtime is bound in Ranked mode so the host-Node runtime is
// materialized instead of leaving the provider bound in custom mode.
func PromoteRuntimeProvidersToRanked(catalog plugincatalog.PluginCatalog, slots map[string]string, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	wc := domain.WizardConfig{
		DiscoveryProvider:  slots[string(domain.SlotDiscovery)],
		RefinementProvider: slots[string(domain.SlotRefinement)],
		ExecutionProvider:  slots[string(domain.SlotExecution)],
	}
	for slot, providerID := range slots {
		if ProviderNeedsRankedRuntime(catalog, providerID) {
			SetRankedSlotMode(&wc, slot)
		}
	}
	return ApplyRankedBindingChoices(catalog, wc, lockFile)
}

// ProviderNeedsRankedRuntime reports whether the provider is certified, ranked
// and declares a runtime that must be materialized.
func ProviderNeedsRankedRuntime(catalog plugincatalog.PluginCatalog, providerID string) bool {
	provider, ok := plugincatalog.FindCatalogProviderRef(catalog, providerID)
	if !ok || !provider.Ranked || provider.CertificationDigest == "" {
		return false
	}
	return domain.NormalizeRankedRuntime(provider.Runtime).Kind != domain.RankedRuntimeNone
}

// SetRankedSlotMode sets the binding mode of slot in wc to Ranked.
func SetRankedSlotMode(wc *domain.WizardConfig, slot string) {
	switch slot {
	case string(domain.SlotDiscovery):
		wc.DiscoveryMode = domain.SlotBindingModeRanked
	case string(domain.SlotRefinement):
		wc.RefinementMode = domain.SlotBindingModeRanked
	case string(domain.SlotExecution):
		wc.ExecutionMode = domain.SlotBindingModeRanked
	}
}

package install

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// promoteRuntimeProvidersToRanked gives a silent install the same outcome the
// wizard's pre-selected Ranked option gives: every certified provider that
// declares a runtime is bound in Ranked mode so the host-Node runtime is
// materialized instead of leaving the provider bound in custom mode.
func promoteRuntimeProvidersToRanked(catalog pluginCatalog, slots map[string]string, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	wc := domain.WizardConfig{
		DiscoveryProvider:  slots["discovery"],
		RefinementProvider: slots["refinement"],
		ExecutionProvider:  slots["execution"],
	}
	for slot, providerID := range slots {
		if providerNeedsRankedRuntime(catalog, providerID) {
			setRankedSlotMode(&wc, slot)
		}
	}
	return applyRankedBindingChoices(catalog, wc, lockFile)
}

// providerNeedsRankedRuntime reports whether the provider is certified, ranked
// and declares a runtime that must be materialized.
func providerNeedsRankedRuntime(catalog pluginCatalog, providerID string) bool {
	provider, ok := findCatalogProvider(catalog, providerID)
	if !ok || !provider.Ranked || provider.CertificationDigest == "" {
		return false
	}
	return domain.NormalizeRankedRuntime(provider.Runtime).Kind != domain.RankedRuntimeNone
}

func setRankedSlotMode(wc *domain.WizardConfig, slot string) {
	switch slot {
	case "discovery":
		wc.DiscoveryMode = domain.SlotBindingModeRanked
	case "refinement":
		wc.RefinementMode = domain.SlotBindingModeRanked
	case "execution":
		wc.ExecutionMode = domain.SlotBindingModeRanked
	}
}

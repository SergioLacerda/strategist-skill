package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func catalogInstallableDefaultProviders(catalog pluginCatalog) map[string]string {
	installable := map[string]string{}
	for _, provider := range catalog.Providers {
		if provider.Installable && provider.LegacyManifestPath != "" {
			installable[provider.ID] = provider.LegacyManifestPath
		}
	}
	return installable
}

// resolveInstallableDefaultProviders returns the provider -> legacy-manifest-path
// map used to decide which providers get a written skill.yaml on install. It
// propagates a loadPluginCatalog failure instead of silently substituting
// installableDefaultProviders (ADR-0035 Decision 2: no fallback substitution).
// In practice this error branch is unreachable via either of this function's
// two current callers: runWizard (wizard.go) and
// activateSilentRoleProviderBindings (installer_silent_role_bindings.go) both
// already hard-block on a loadPluginCatalog failure earlier in the same call
// stack — the same "unreachable in practice" position as loadKnownProviders
// (wizard_fallback_providers.go). Propagating here is defense in depth against
// a future caller that lacks that upstream guard, not a fix for a currently
// reachable bug.
//
// A successfully loaded catalog with zero installable-flagged entries is not
// an error case and still falls back to installableDefaultProviders.
func resolveInstallableDefaultProviders(extractor domain.FileExtractor) (map[string]string, error) {
	catalog, err := loadPluginCatalog(extractor)
	if err != nil {
		return nil, fmt.Errorf("resolve installable default providers: %w", err)
	}
	installable := catalogInstallableDefaultProviders(catalog)
	if len(installable) == 0 {
		return installableDefaultProviders, nil
	}
	return installable, nil
}

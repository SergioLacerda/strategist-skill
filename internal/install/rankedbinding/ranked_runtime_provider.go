package rankedbinding

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/filesystem"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
)

// ResolveRankedProvider finds the bound catalog provider and its validated
// runtime contract; the provider must be certified.
func ResolveRankedProvider(catalog plugincatalog.PluginCatalog, binding domain.SlotBinding) (plugincatalog.Provider, domain.WeaponRuntime, error) {
	provider, ok := plugincatalog.FindCatalogProvider(catalog, binding.InstalledInstanceID)
	if !ok {
		return provider, domain.WeaponRuntime{}, fmt.Errorf("ranked runtime provider %q is missing from catalog", binding.InstalledInstanceID)
	}
	if !provider.Ranked || provider.CertificationDigest == "" {
		return provider, domain.WeaponRuntime{}, fmt.Errorf("ranked runtime provider %q is not certified", provider.ID)
	}
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	if err := runtime.Validate(); err != nil {
		return provider, runtime, fmt.Errorf("ranked runtime provider %q: %w", provider.ID, err)
	}
	return provider, runtime, nil
}

// BootstrapRankedProvider resolves the executable, bootstraps the runtime root
// and returns the private-runtime evidence (nil for a host executable).
func BootstrapRankedProvider(ctx context.Context, strategistDir string, provider plugincatalog.Provider, runtime domain.WeaponRuntime) (*domain.RankedRuntimeStateRuntime, error) {
	if runtime.Kind != domain.RankedRuntimeOpenSpecRoot {
		return nil, nil
	}
	root, err := filesystem.SafeJoinExisting(strategistDir, filepath.ToSlash(runtime.Root)[len(".strategist/"):])
	if err != nil {
		return nil, fmt.Errorf("ranked runtime provider %q root: %w", provider.ID, err)
	}
	exe, private, err := ResolveRankedExecutable(ctx, strategistDir, provider.ID, provider.Version, runtime)
	if err != nil {
		return nil, rankedRuntimeBootstrapError(provider.ID, err)
	}
	if err := BootstrapOpenSpecRuntimeWith(ctx, root, runtime, exe); err != nil {
		return nil, rankedRuntimeBootstrapError(provider.ID, err)
	}
	return private, nil
}

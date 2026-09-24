package install

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
)

// resolveRankedProvider finds the bound catalog provider and its validated
// runtime contract; the provider must be certified.
func resolveRankedProvider(catalog pluginCatalog, binding domain.SlotBinding) (pluginCatalogProvider, domain.RankedRuntimeContract, error) {
	provider, ok := findCatalogProvider(catalog, binding.InstalledInstanceID)
	if !ok {
		return provider, domain.RankedRuntimeContract{}, fmt.Errorf("ranked runtime provider %q is missing from catalog", binding.InstalledInstanceID)
	}
	if !provider.Ranked || provider.CertificationDigest == "" {
		return provider, domain.RankedRuntimeContract{}, fmt.Errorf("ranked runtime provider %q is not certified", provider.ID)
	}
	runtime := domain.NormalizeRankedRuntime(provider.Runtime)
	if err := runtime.Validate(); err != nil {
		return provider, runtime, fmt.Errorf("ranked runtime provider %q: %w", provider.ID, err)
	}
	return provider, runtime, nil
}

// bootstrapRankedProvider resolves the executable, bootstraps the runtime root
// and returns the private-runtime evidence (nil for a host executable).
func bootstrapRankedProvider(ctx context.Context, strategistDir string, provider pluginCatalogProvider, runtime domain.RankedRuntimeContract) (*domain.RankedRuntimeStateRuntime, error) {
	if runtime.Kind != domain.RankedRuntimeOpenSpecRoot {
		return nil, nil
	}
	root, err := runtimefs.SafeJoinExisting(strategistDir, filepath.ToSlash(runtime.Root)[len(".strategist/"):])
	if err != nil {
		return nil, fmt.Errorf("ranked runtime provider %q root: %w", provider.ID, err)
	}
	exe, private, err := resolveRankedExecutable(ctx, strategistDir, provider.ID, runtime)
	if err != nil {
		return nil, rankedRuntimeBootstrapError(provider.ID, err)
	}
	if err := bootstrapOpenSpecRuntimeWith(ctx, root, runtime, exe); err != nil {
		return nil, rankedRuntimeBootstrapError(provider.ID, err)
	}
	return private, nil
}

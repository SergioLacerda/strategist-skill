package install

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// customProviderResolution is the install-time evidence needed to bind a
// typed Custom provider without promoting it into the embedded catalog.
type customProviderResolution struct {
	Package connectors.ResolvedProviderPackage
	Slot    string
}

// catalogWithCustomProviders projects explicitly typed, non-catalog Custom
// providers into the current onboarding plan. The projection exists only for
// this plan; the selected package remains external and the canonical catalog
// is never mutated.
func catalogWithCustomProviders(catalog pluginCatalog, slots, modes map[string]string) (pluginCatalog, map[string]customProviderResolution, error) {
	if len(modes) == 0 {
		return catalog, nil, nil
	}
	env, err := currentCustomProviderEnv()
	if err != nil {
		return pluginCatalog{}, nil, err
	}
	return projectCustomProviders(env, catalog, slots, modes)
}

func projectCustomProviders(env customProviderEnv, catalog pluginCatalog, slots, modes map[string]string) (pluginCatalog, map[string]customProviderResolution, error) {
	resolved := catalog
	providers := make(map[string]customProviderResolution)
	for _, slot := range sortedSlotNames(slots) {
		providerID := slots[slot]
		if !needsCustomProjection(resolved, providerID, modes[slot]) {
			continue
		}
		provider, resolution, err := resolveCustomSlotProvider(env, providerID, slot)
		if err != nil {
			return pluginCatalog{}, nil, err
		}
		resolved.Providers = append(resolved.Providers, provider)
		providers[providerID] = resolution
	}
	return resolved, providers, nil
}

// needsCustomProjection reports whether providerID is an explicitly typed
// Custom provider that the catalog does not already know.
func needsCustomProjection(catalog pluginCatalog, providerID, mode string) bool {
	if mode != domain.SlotBindingModeCustom || providerID == "" {
		return false
	}
	_, known := findCatalogProvider(catalog, providerID)
	return !known
}

// customProviderEnv is the workspace and global-root context Custom provider
// resolution depends on.
type customProviderEnv struct {
	workspaceRoot string
	globalRoots   []connectors.ProviderRoot
}

func currentCustomProviderEnv() (customProviderEnv, error) {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return customProviderEnv{}, fmt.Errorf("resolve custom provider workspace: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return customProviderEnv{}, fmt.Errorf("resolve custom provider home: %w", err)
	}
	return customProviderEnv{workspaceRoot: workspaceRoot, globalRoots: connectors.DefaultGlobalProviderRoots(homeDir)}, nil
}

func resolveCustomSlotProvider(env customProviderEnv, providerID, slot string) (pluginCatalogProvider, customProviderResolution, error) {
	packageEvidence, err := connectors.ResolveCustomProviderPackage(env.workspaceRoot, providerID, env.globalRoots)
	if err != nil {
		return pluginCatalogProvider{}, customProviderResolution{}, fmt.Errorf("resolve Custom provider %q for slot %s: %w", providerID, slot, err)
	}
	provider := customCatalogProvider(packageEvidence, slot)
	if err := validateCatalogProvider(provider); err != nil {
		return pluginCatalogProvider{}, customProviderResolution{}, fmt.Errorf("custom provider %q for slot %s: %w", providerID, slot, err)
	}
	return provider, customProviderResolution{Package: packageEvidence, Slot: slot}, nil
}

func customCatalogProvider(evidence connectors.ResolvedProviderPackage, slot string) pluginCatalogProvider {
	role := slotRoleID(domain.SlotName(slot))
	return pluginCatalogProvider{
		ID: evidence.Package.ID, Version: evidence.Package.Version, SchemaVersion: "1",
		Kind: domain.WeaponKindAtomic, Origin: domain.WeaponOriginCustom, Status: "active", RiskScore: customRiskForSlot(slot),
		CanonicalRole: role, Roles: []string{role}, SupportedSlots: []string{slot},
		SupportedHandoffSchemas: []string{slotHandoffSchema(domain.SlotName(slot))},
		Installable:             true, CompatibilitySource: "external", PackageDigest: evidence.Package.Digest,
		Runtime: domain.RankedRuntimeContract{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
		WeaponContract: domain.WeaponContract{
			RoleOwner: role, Participation: "required", InvocationEvidence: "required",
			UnavailableBehavior: "role_invocation_failed", NativeSubstitution: "forbidden",
		},
	}
}

func customRiskForSlot(slot string) string {
	if slot == string(domain.SlotExecution) {
		return "controlled"
	}
	return "write_analysis"
}

func annotateCustomProviderInstances(lockFile *domain.PluginLockFile, providers map[string]customProviderResolution) {
	for providerID, resolution := range providers {
		for i := range lockFile.Inventory.Instances {
			if lockFile.Inventory.Instances[i].ID != providerID {
				continue
			}
			instance := &lockFile.Inventory.Instances[i]
			instance.PackageDigest = resolution.Package.Package.Digest
			instance.ConnectorID = "host"
			instance.ProviderOrigin = resolution.Package.Origin
			instance.SeedPath = resolution.Package.SeedPath
			instance.Entrypoint = resolution.Package.Entrypoint
		}
	}
}

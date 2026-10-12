package customprovider

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// Resolution is the install-time evidence needed to bind a
// typed Custom provider without promoting it into the embedded catalog.
type Resolution struct {
	Package connectors.ResolvedProviderPackage
	Slot    string
	// Declaration and Version come from the package's Strategist sidecar; they
	// are what normalization writes into the installed package representation.
	Declaration plugincatalog.ExternalSkillAdapter
	Version     string
}

// CatalogWithCustomProviders projects explicitly typed, non-catalog Custom
// providers into the current onboarding plan. The projection exists only for
// this plan; the selected package remains external and the canonical catalog
// is never mutated.
func CatalogWithCustomProviders(catalog plugincatalog.PluginCatalog, slots, modes map[string]string) (plugincatalog.PluginCatalog, map[string]Resolution, error) {
	if len(modes) == 0 {
		return catalog, nil, nil
	}
	env, err := currentCustomProviderEnv()
	if err != nil {
		return plugincatalog.PluginCatalog{}, nil, err
	}
	return ProjectCustomProviders(env, catalog, slots, modes)
}

// ProjectCustomProviders returns catalog extended with the Custom providers selected for the slots that need a
// projection, together with their resolutions.
func ProjectCustomProviders(env Env, catalog plugincatalog.PluginCatalog, slots, modes map[string]string) (plugincatalog.PluginCatalog, map[string]Resolution, error) {
	resolved := catalog
	providers := make(map[string]Resolution)
	for _, slot := range support.SortedSlotNames(slots) {
		providerID := slots[slot]
		if !NeedsCustomProjection(resolved, providerID, modes[slot]) {
			continue
		}
		provider, resolution, err := ResolveCustomSlotProvider(env, providerID, slot)
		if err != nil {
			return plugincatalog.PluginCatalog{}, nil, err
		}
		resolved.Providers = append(resolved.Providers, provider)
		providers[providerID] = resolution
	}
	return resolved, providers, nil
}

// NeedsCustomProjection reports whether providerID is an explicitly typed
// Custom provider that the catalog does not already know.
func NeedsCustomProjection(catalog plugincatalog.PluginCatalog, providerID, mode string) bool {
	if mode != domain.SlotBindingModeCustom || providerID == "" {
		return false
	}
	return !plugincatalog.CatalogKnowsRef(catalog, providerID)
}

// Env is the workspace and global-root context Custom provider
// resolution depends on.
type Env struct {
	workspaceRoot string
	globalRoots   []connectors.ProviderRoot
}

func currentCustomProviderEnv() (Env, error) {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return Env{}, fmt.Errorf("resolve custom provider workspace: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Env{}, fmt.Errorf("resolve custom provider home: %w", err)
	}
	return Env{workspaceRoot: workspaceRoot, globalRoots: connectors.DefaultGlobalProviderRoots(homeDir)}, nil
}

// ResolveCustomSlotProvider resolves the Custom provider selected for slot against env.
func ResolveCustomSlotProvider(env Env, providerID, slot string) (plugincatalog.Provider, Resolution, error) {
	packageEvidence, err := connectors.ResolveCustomProviderPackage(env.workspaceRoot, providerID, env.globalRoots)
	if err != nil {
		return plugincatalog.Provider{}, Resolution{}, fmt.Errorf("resolve Custom provider %q for slot %s: %w", providerID, slot, err)
	}
	declaration, version, err := DeclaredCustomPackage(packageEvidence, slot)
	if err != nil {
		return plugincatalog.Provider{}, Resolution{}, fmt.Errorf("custom provider %q for slot %s: %w", providerID, slot, err)
	}
	provider := CustomCatalogProvider(packageEvidence, slot, version, declaration.RiskScore)
	if err := plugincatalog.ValidateCatalogProvider(provider); err != nil {
		return plugincatalog.Provider{}, Resolution{}, fmt.Errorf("custom provider %q for slot %s: %w", providerID, slot, err)
	}
	return provider, Resolution{Package: packageEvidence, Slot: slot, Declaration: declaration, Version: version}, nil
}

// CustomCatalogProvider builds the catalog entry that represents a resolved Custom provider package.
func CustomCatalogProvider(evidence connectors.ResolvedProviderPackage, slot, version, riskScore string) plugincatalog.Provider {
	role := plugincatalog.SlotRoleID(domain.SlotName(slot))
	return plugincatalog.Provider{
		ID: evidence.Package.ID, Version: version, SchemaVersion: "1",
		Kind: domain.WeaponKindAtomic, Origin: domain.WeaponOriginCustom, Status: "active", RiskScore: riskScore,
		CanonicalRole: role, Roles: []string{role}, SupportedSlots: []string{slot},
		SupportedHandoffSchemas: []string{plugincatalog.SlotHandoffSchema(domain.SlotName(slot))},
		Installable:             true, CompatibilitySource: names.CompatibilityExternal, PackageDigest: evidence.Package.Digest,
		Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
		WeaponContract: domain.WeaponContract{
			RoleOwner: role, Participation: "required", InvocationEvidence: "required",
			UnavailableBehavior: "role_invocation_failed", NativeSubstitution: "forbidden",
		},
	}
}

// AnnotateCustomProviderInstances annotates the installed instances of lock with the Custom package they were bound to.
func AnnotateCustomProviderInstances(lockFile *domain.PluginLockFile, providers map[string]Resolution) {
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

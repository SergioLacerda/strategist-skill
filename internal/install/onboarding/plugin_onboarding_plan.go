package onboarding

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application/installplan"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
)

// PlanPluginOnboardingWithModes resolves the onboarding plan for slots and modes against catalog.
func PlanPluginOnboardingWithModes(extractor domain.FileExtractor, catalog plugincatalog.PluginCatalog, slots, modes map[string]string) (PluginOnboardingPlan, error) {
	inputs, err := resolvePluginOnboardingInputs(catalog, slots, modes)
	if err != nil {
		return PluginOnboardingPlan{}, err
	}
	roleMigration, err := PlanRoleProviderMigrationWithCatalog(extractor, inputs.Catalog, slots)
	if err != nil {
		return PluginOnboardingPlan{}, fmt.Errorf("resolve role/provider bindings: %w", err)
	}
	lock := appendRoleMigrationToLock(inputs.Lock, roleMigration)
	roster, selections, err := buildRosterArtifacts(roleMigration)
	if err != nil {
		return PluginOnboardingPlan{}, fmt.Errorf("build Weapon Roster: %w", err)
	}
	installPlan, err := installplan.PlanInstall(installplan.Input{
		Stage: domain.StageRoster, Slots: slots, SlotModes: modes,
		Lock: lock, Bindings: inputs.Bindings,
	})
	if err != nil {
		return PluginOnboardingPlan{}, fmt.Errorf("build install plan: %w", err)
	}

	return PluginOnboardingPlan{
		SchemaVersion:        "strategist-plugin-onboarding-plan/v1",
		RequiresConfirmation: true,
		InstallPlan:          installPlan,
		Options:              inputs.Options,
		Roster:               roster,
		Selections:           selections,
		Lock:                 lock,
		Inventory:            domain.PluginInventory{SchemaVersion: "strategist-plugin-inventory/v1", Instances: inputs.Instances},
		Bindings:             inputs.Bindings,
		Changes:              changesFromBindings(inputs.Bindings),
		RoleMigration:        roleMigration,
		CustomProviders:      inputs.CustomProviders,
	}, nil
}

type pluginOnboardingInputs struct {
	Catalog         plugincatalog.PluginCatalog
	CustomProviders map[string]customprovider.Resolution
	Options         plugincatalog.InstallOptionSet
	Lock            domain.PluginLock
	Instances       []domain.InstalledInstance
	Bindings        []domain.SlotBinding
}

func resolvePluginOnboardingInputs(catalog plugincatalog.PluginCatalog, slots, modes map[string]string) (pluginOnboardingInputs, error) {
	resolvedCatalog, customProviders, err := customprovider.CatalogWithCustomProviders(catalog, slots, modes)
	if err != nil {
		return pluginOnboardingInputs{}, err //nolint:wrapcheck // sibling package of the same module; its error already names the provider
	}
	optionSet := plugincatalog.NewInstallOptionSet(resolvedCatalog)
	if err := optionSet.ValidateRankedSelections(slots, modes); err != nil {
		return pluginOnboardingInputs{}, err //nolint:wrapcheck // sibling package of the same module; its error already names the selection
	}
	requirements, err := Requirements(resolvedCatalog, slots)
	if err != nil {
		return pluginOnboardingInputs{}, err
	}
	lock, err := resolver.Resolve(requirements, plugincatalog.CatalogResolverCandidates(resolvedCatalog))
	if err != nil {
		return pluginOnboardingInputs{}, fmt.Errorf("resolve plugin lock: %w", err)
	}
	bindings, err := BindingsFromSlots(slots, lock)
	if err != nil {
		return pluginOnboardingInputs{}, err
	}
	return pluginOnboardingInputs{
		Catalog: resolvedCatalog, CustomProviders: customProviders, Options: optionSet,
		Lock: lock, Instances: inventoryFromLock(lock), Bindings: bindings,
	}, nil
}

func appendRoleMigrationToLock(lock domain.PluginLock, migration RoleProviderMigrationPreview) domain.PluginLock {
	lock.Nodes = appendRoleMigrationNodes(lock.Nodes, migration)
	lock.GraphDigest = resolver.DigestLockNodes(lock.Nodes)
	lock.ResolutionID = lock.GraphDigest
	return lock
}

// Requirements returns the resolver requirements implied by the chosen slots.
func Requirements(catalog plugincatalog.PluginCatalog, slots map[string]string) ([]resolver.Requirement, error) {
	requirements := make([]resolver.Requirement, 0, len(slots))
	for _, slot := range support.SortedSlotNames(slots) {
		provider := slots[slot]
		if provider == "" {
			return nil, fmt.Errorf("unresolved_active_slot: %s has empty provider", slot)
		}
		resolved, ok := plugincatalog.FindCatalogProviderRef(catalog, provider)
		if !ok {
			return nil, fmt.Errorf("unresolved_active_slot: %s provider %s%s", slot, provider, plugincatalog.UnresolvedRefHint(catalog, provider))
		}
		// The requirement pins the resolved id@version, so the resolver can never
		// pick "the highest" of several catalogued versions.
		requirements = append(requirements, resolver.Requirement{ID: resolved.ID, Kind: "adapter_contract", Constraint: plugincatalog.ProviderVersionOrDefault(resolved.Version)})
	}
	return requirements, nil
}

func appendRoleMigrationNodes(nodes []domain.PluginLockNode, migration RoleProviderMigrationPreview) []domain.PluginLockNode {
	for _, entry := range migration.Entries {
		if entry.ResolutionError == "" {
			nodes = append(nodes, plugins.RoleBindingLockNode(entry.Resolved))
		}
	}
	return nodes
}

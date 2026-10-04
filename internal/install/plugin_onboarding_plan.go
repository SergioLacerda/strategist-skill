package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
)

func planPluginOnboardingWithModes(extractor domain.FileExtractor, catalog pluginCatalog, slots, modes map[string]string) (pluginOnboardingPlan, error) {
	inputs, err := resolvePluginOnboardingInputs(catalog, slots, modes)
	if err != nil {
		return pluginOnboardingPlan{}, err
	}
	roleMigration, err := planRoleProviderMigrationWithCatalog(extractor, inputs.Catalog, slots)
	if err != nil {
		return pluginOnboardingPlan{}, fmt.Errorf("resolve role/provider bindings: %w", err)
	}
	lock := appendRoleMigrationToLock(inputs.Lock, roleMigration)
	roster, selections, err := buildRosterArtifacts(roleMigration)
	if err != nil {
		return pluginOnboardingPlan{}, fmt.Errorf("build Weapon Roster: %w", err)
	}
	installPlan, err := application.PlanInstall(application.InstallPlanInput{
		Stage: domain.StageRoster, Slots: slots, SlotModes: modes,
		Lock: lock, Bindings: inputs.Bindings,
	})
	if err != nil {
		return pluginOnboardingPlan{}, fmt.Errorf("build install plan: %w", err)
	}

	return pluginOnboardingPlan{
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
	Catalog         pluginCatalog
	CustomProviders map[string]customProviderResolution
	Options         installOptionSet
	Lock            domain.PluginLock
	Instances       []domain.InstalledInstance
	Bindings        []domain.SlotBinding
}

func resolvePluginOnboardingInputs(catalog pluginCatalog, slots, modes map[string]string) (pluginOnboardingInputs, error) {
	resolvedCatalog, customProviders, err := catalogWithCustomProviders(catalog, slots, modes)
	if err != nil {
		return pluginOnboardingInputs{}, err
	}
	optionSet := newInstallOptionSet(resolvedCatalog)
	if err := optionSet.ValidateRankedSelections(slots, modes); err != nil {
		return pluginOnboardingInputs{}, err
	}
	requirements, err := onboardingRequirements(resolvedCatalog, slots)
	if err != nil {
		return pluginOnboardingInputs{}, err
	}
	lock, err := plugins.Resolve(requirements, catalogResolverCandidates(resolvedCatalog))
	if err != nil {
		return pluginOnboardingInputs{}, fmt.Errorf("resolve plugin lock: %w", err)
	}
	bindings, err := bindingsFromSlots(slots, lock)
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
	lock.GraphDigest = plugins.DigestLockNodes(lock.Nodes)
	lock.ResolutionID = lock.GraphDigest
	return lock
}

func onboardingRequirements(catalog pluginCatalog, slots map[string]string) ([]plugins.Requirement, error) {
	requirements := make([]plugins.Requirement, 0, len(slots))
	for _, slot := range sortedSlotNames(slots) {
		provider := slots[slot]
		if provider == "" {
			return nil, fmt.Errorf("unresolved_active_slot: %s has empty provider", slot)
		}
		resolved, ok := findCatalogProviderRef(catalog, provider)
		if !ok {
			return nil, fmt.Errorf("unresolved_active_slot: %s provider %s%s", slot, provider, unresolvedRefHint(catalog, provider))
		}
		// The requirement pins the resolved id@version, so the resolver can never
		// pick "the highest" of several catalogued versions.
		requirements = append(requirements, plugins.Requirement{ID: resolved.ID, Kind: "adapter_contract", Constraint: providerVersionOrDefault(resolved.Version)})
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

package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

const roleSlotMapPath = "roles/default.yaml"

// RoleProviderPreviewEntry, RoleProviderMigrationPreview, and its
// FullyResolved/Evidence/Preview methods live in
// role_provider_migration_preview.go, split out to keep this file under the
// repo's file-size budget.

// PlanRoleProviderMigration builds the preview for the given legacy
// active.slots map, reading the canonical slot->role mapping
// (roles/default.yaml) and per-role contracts (roles/<name>.yaml) plus the
// plugin catalog through extractor — the same embedded-defaults source
// runWizard already reads from, so this never touches the runtime
// .strategist/ tree (FileExtractor.ReadFile is write-only there).
func PlanRoleProviderMigration(extractor domain.FileExtractor, activeSlots map[string]string) (RoleProviderMigrationPreview, error) {
	roleSlotMap, err := loadRoleSlotMap(extractor)
	if err != nil {
		return RoleProviderMigrationPreview{}, fmt.Errorf("role/provider migration: %w", err)
	}
	catalog, err := loadPluginCatalog(extractor)
	if err != nil {
		return RoleProviderMigrationPreview{}, fmt.Errorf("role/provider migration: %w", err)
	}

	entries := make([]RoleProviderPreviewEntry, 0, len(domain.RequiredSlots()))
	for _, slot := range domain.RequiredSlots() {
		entry, err := planRoleProviderEntry(extractor, catalog, roleSlotMap, activeSlots, slot)
		if err != nil {
			return RoleProviderMigrationPreview{}, err
		}
		entries = append(entries, entry)
	}
	return RoleProviderMigrationPreview{Entries: entries}, nil
}

func planRoleProviderEntry(extractor domain.FileExtractor, catalog pluginCatalog, roleSlotMap map[string]string, activeSlots map[string]string, slot domain.SlotName) (RoleProviderPreviewEntry, error) {
	slotName := string(slot)
	roleName := roleSlotMap[slotName]
	roleCfg, err := loadRoleConfig(extractor, roleName)
	if err != nil {
		return RoleProviderPreviewEntry{}, fmt.Errorf("role/provider migration: slot %s: %w", slotName, err)
	}
	role := domain.RoleContractFromConfig(roleCfg, domain.RoleHandoffSchema[roleName])
	candidates := providerContractsForRole(catalog, roleName)
	entry := RoleProviderPreviewEntry{Slot: slotName, RoleName: roleName, CurrentProviderID: activeSlots[slotName], Candidates: candidates}
	if current := activeSlots[slotName]; current != "" {
		if _, ok := findCatalogProvider(catalog, current); !ok {
			entry.ResolutionError = fmt.Sprintf("unresolved_active_slot: %s provider %s not found in catalog", slotName, current)
			return entry, nil
		}
	}
	binding, resolveErr := plugins.ResolveRoleBinding(role, candidates, "", activeSlots[slotName])
	if resolveErr != nil {
		entry.ResolutionError = resolveErr.Error()
	} else {
		entry.Resolved = binding
	}
	return entry, nil
}

// ApplyRoleProviderMigration activates the resolved Provider for the
// discovery/refinement entries as each slot's live plugin binding, reusing the
// exact lifecycle.Store staged activation (Begin/Stage/Probe/Activate) and last-known-good rollback
// primitives applyPluginOnboardingPlan already uses for legacy slot bindings
// — no second activation mechanism is introduced (Decision 4: lifecycle
// reuse; tasks.md Task 3.3/4.2).
//
// It refuses to run at all unless preview.FullyResolved(): Task 4.2 requires
// "apply only a fully resolved binding," and there is no safe partial-apply
// story where some slots move to their new Provider while others silently
// keep failing to resolve.
func ApplyRoleProviderMigration(store *lifecycle.Store, preview RoleProviderMigrationPreview, probe pluginProbeFunc) error {
	if !preview.FullyResolved() {
		return fmt.Errorf("role_provider_migration_not_fully_resolved: refusing to apply a partial migration")
	}
	for _, entry := range preview.Entries {
		if !isPersistedRoleProviderSlot(entry.Slot) {
			continue
		}
		desired := domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			Slot:                entry.Slot,
			InstalledInstanceID: entry.Resolved.Provider.ID,
			Status:              "enabled",
		}
		if err := applyPluginBinding(store, desired, probe); err != nil {
			return fmt.Errorf("apply role/provider migration: slot %s: %w", entry.Slot, err)
		}
	}
	return nil
}

// activateRoleProviderMigration drives preview's resolved bindings through a
// real lifecycle.Store — Begin/Stage/Probe/Activate per slot, with automatic
// rollback on probe failure — closing the "zero production callers" gap
// ApplyRoleProviderMigration otherwise has (tasks.md Task 5,
// .analysis/refined/20260913-embedded-skill-directory-catalog).
//
// The store is seeded from plugins.lock under strategistDir when one exists
// (docs/adr/0037-wizard-role-binding-persistence.md, DEC-001), so a resolved
// binding survives across `strategist install` invocations instead of being
// rebuilt from scratch and discarded every time
// (.analysis/refined/20260913-wizard-plugin-lifecycle-persistence-gap/). It
// only reads plugins.lock, never writes it — the caller (applyWizardConfig)
// persists the returned state after active.yaml is written successfully, so
// a wizard run that fails before reaching that point never leaves a stray
// plugins.lock with no corresponding active.yaml. An empty strategistDir
// (used by tests that exercise activation mechanics only, not a real
// installation) skips the read — the store is seeded fresh exactly as it
// was before persistence existed. The probe here is intentionally a simple,
// always-successful check: a connector-aware probe policy is a further
// enhancement this task does not claim, matching the same level of rigor
// plugin_onboarding_test.go's own probe closures already use for the legacy
// binding path.
func activateRoleProviderMigration(strategistDir string, resolvedLock domain.PluginLock, preview RoleProviderMigrationPreview) (domain.PluginLockFile, error) {
	var persisted domain.PluginLockFile
	if strategistDir != "" {
		var err error
		persisted, err = readPluginLockFile(strategistDir)
		if err != nil {
			return domain.PluginLockFile{}, fmt.Errorf("activate role/provider migration: %w", err)
		}
	}

	store := lifecycle.NewStore()
	store.Inventory = persisted.Inventory
	store.Bindings = persisted.Bindings

	for _, entry := range preview.Entries {
		if !isPersistedRoleProviderSlot(entry.Slot) {
			continue
		}
		seedRoleProviderMigrationEntry(store, entry)
	}

	probe := func(domain.SlotBinding, domain.InstalledInstance) bool { return true }
	if err := ApplyRoleProviderMigration(store, preview, probe); err != nil {
		return domain.PluginLockFile{}, err
	}

	return domain.PluginLockFile{Lock: resolvedLock, Inventory: store.Inventory, Bindings: store.Bindings}, nil
}

func isPersistedRoleProviderSlot(slot string) bool {
	return slot == string(domain.SlotDiscovery) || slot == string(domain.SlotRefinement)
}

// seedRoleProviderMigrationEntry ensures store has an instance/binding entry
// for entry's current and resolved providers before ApplyRoleProviderMigration
// runs, without duplicating an instance or binding a prior persisted run
// already recorded (docs/adr/0037-wizard-role-binding-persistence.md).
func seedRoleProviderMigrationEntry(store *lifecycle.Store, entry RoleProviderPreviewEntry) {
	if entry.CurrentProviderID != "" {
		if _, ok := store.Instance(entry.CurrentProviderID); !ok {
			store.Inventory.Instances = append(store.Inventory.Instances, domain.InstalledInstance{
				ID: entry.CurrentProviderID, State: lifecycle.StateActive, LastKnownGood: true,
			})
		}
		if _, ok := store.Binding(entry.Slot); !ok {
			store.Bindings = append(store.Bindings, domain.SlotBinding{
				Slot: entry.Slot, InstalledInstanceID: entry.CurrentProviderID, Generation: 1, Status: "enabled",
			})
		}
	}
	if _, ok := store.Instance(entry.Resolved.Provider.ID); !ok {
		store.Inventory.Instances = append(store.Inventory.Instances, domain.InstalledInstance{
			ID: entry.Resolved.Provider.ID, State: lifecycle.StateActive,
		})
	}
}

// loadRoleSlotMap and loadRoleConfig live in role_provider_migration_loaders.go,
// split out to keep this file under the repo's file-size budget.

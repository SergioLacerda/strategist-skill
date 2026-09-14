package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"gopkg.in/yaml.v3"
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
		slotName := string(slot)
		roleName := roleSlotMap[slotName]
		roleCfg, err := loadRoleConfig(extractor, roleName)
		if err != nil {
			return RoleProviderMigrationPreview{}, fmt.Errorf("role/provider migration: slot %s: %w", slotName, err)
		}
		role := domain.RoleContractFromConfig(roleCfg, "")
		candidates := providerContractsForRole(catalog, roleName)

		entry := RoleProviderPreviewEntry{
			Slot:              slotName,
			RoleName:          roleName,
			CurrentProviderID: activeSlots[slotName],
			Candidates:        candidates,
		}
		if current := activeSlots[slotName]; current != "" {
			if _, ok := findCatalogProvider(catalog, current); !ok {
				entry.ResolutionError = fmt.Sprintf("unresolved_active_slot: %s provider %s not found in catalog", slotName, current)
				entries = append(entries, entry)
				continue
			}
		}
		binding, resolveErr := plugins.ResolveRoleBinding(role, candidates, "", activeSlots[slotName])
		if resolveErr != nil {
			entry.ResolutionError = resolveErr.Error()
		} else {
			entry.Resolved = binding
		}
		entries = append(entries, entry)
	}
	return RoleProviderMigrationPreview{Entries: entries}, nil
}

// ApplyRoleProviderMigration activates every entry's resolved Provider as its
// slot's live plugin binding, reusing the exact lifecycle.Store staged
// activation (Begin/Stage/Probe/Activate) and last-known-good rollback
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
// real, ephemeral lifecycle.Store — Begin/Stage/Probe/Activate per slot, with
// automatic rollback on probe failure — closing the "zero production
// callers" gap ApplyRoleProviderMigration otherwise has (tasks.md Task 5,
// .analysis/refined/20260913-embedded-skill-directory-catalog).
//
// The store is seeded fresh from preview's own current/resolved provider ids
// for this single wizard run; it is not persisted across invocations
// (persisting SlotBinding/PluginLock across runs is a separate, larger gap —
// see .analysis/done/20260913-wizard-plugin-lifecycle-persistence-gap-analysis.md
// — this task closes only the "nothing ever calls Apply*" half of it). The
// probe here is intentionally a simple, always-successful check: a
// connector-aware probe policy is a further enhancement this task does not
// claim, matching the same level of rigor plugin_onboarding_test.go's own
// probe closures already use for the legacy binding path.
func activateRoleProviderMigration(preview RoleProviderMigrationPreview) error {
	store := lifecycle.NewStore()
	for _, entry := range preview.Entries {
		if entry.CurrentProviderID != "" {
			store.Inventory.Instances = append(store.Inventory.Instances, domain.InstalledInstance{
				ID: entry.CurrentProviderID, State: lifecycle.StateActive, LastKnownGood: true,
			})
			store.Bindings = append(store.Bindings, domain.SlotBinding{
				Slot: entry.Slot, InstalledInstanceID: entry.CurrentProviderID, Generation: 1, Status: "enabled",
			})
		}
		store.Inventory.Instances = append(store.Inventory.Instances, domain.InstalledInstance{
			ID: entry.Resolved.Provider.ID, State: lifecycle.StateActive,
		})
	}

	probe := func(domain.SlotBinding, domain.InstalledInstance) bool { return true }
	return ApplyRoleProviderMigration(store, preview, probe)
}

func loadRoleSlotMap(extractor domain.FileExtractor) (domain.RoleSlotMap, error) {
	data, err := extractor.ReadFile(roleSlotMapPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", roleSlotMapPath, err)
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(data, &roleMap); err != nil {
		return nil, fmt.Errorf("%s: %w", roleSlotMapPath, err)
	}
	if err := roleMap.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", roleSlotMapPath, err)
	}
	return roleMap, nil
}

func loadRoleConfig(extractor domain.FileExtractor, roleName string) (domain.RoleConfig, error) {
	path := "roles/" + roleName + ".yaml"
	data, err := extractor.ReadFile(path)
	if err != nil {
		return domain.RoleConfig{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg domain.RoleConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

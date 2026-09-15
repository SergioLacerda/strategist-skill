package install

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type pluginOnboardingPlan struct {
	SchemaVersion        string
	RequiresConfirmation bool
	Lock                 domain.PluginLock
	Inventory            domain.PluginInventory
	Bindings             []domain.SlotBinding
	Changes              []string
	// RoleMigration is the Role/Provider convergence preview for the same
	// slots (tasks.md Task 4.1/4.2, strategist-papeis-personagens-skills-nativas).
	// It is additive evidence over the legacy Lock/Bindings above, never a
	// replacement for them (Decision 4: lifecycle reuse) — a role/provider
	// resolution failure for one slot never fails plugin onboarding as a
	// whole, since the legacy catalog/lock resolution above already is the
	// authoritative gate for whether a slot's provider is usable.
	RoleMigration RoleProviderMigrationPreview
}

type pluginProbeFunc func(domain.SlotBinding, domain.InstalledInstance) bool

func planPluginOnboarding(extractor domain.FileExtractor, catalog pluginCatalog, slots map[string]string) (pluginOnboardingPlan, error) {
	requirements := make([]plugins.Requirement, 0, len(slots))
	for _, slot := range sortedSlotNames(slots) {
		provider := slots[slot]
		if provider == "" {
			return pluginOnboardingPlan{}, fmt.Errorf("unresolved_active_slot: %s has empty provider", slot)
		}
		if _, ok := findCatalogProvider(catalog, provider); !ok {
			return pluginOnboardingPlan{}, fmt.Errorf("unresolved_active_slot: %s provider %s", slot, provider)
		}
		requirements = append(requirements, plugins.Requirement{ID: provider, Kind: "adapter_contract", Constraint: "*"})
	}

	lock, err := plugins.Resolve(requirements, catalogResolverCandidates(catalog))
	if err != nil {
		return pluginOnboardingPlan{}, fmt.Errorf("resolve plugin lock: %w", err)
	}
	instances := inventoryFromLock(lock)
	bindings, err := bindingsFromSlots(slots, lock)
	if err != nil {
		return pluginOnboardingPlan{}, err
	}
	changes := changesFromBindings(bindings)

	roleMigration, err := PlanRoleProviderMigration(extractor, slots)
	if err != nil {
		return pluginOnboardingPlan{}, fmt.Errorf("resolve role/provider bindings: %w", err)
	}
	for _, entry := range roleMigration.Entries {
		if entry.ResolutionError != "" {
			continue
		}
		lock.Nodes = append(lock.Nodes, plugins.RoleBindingLockNode(entry.Resolved))
	}
	lock.GraphDigest = plugins.DigestLockNodes(lock.Nodes)
	lock.ResolutionID = lock.GraphDigest

	return pluginOnboardingPlan{
		SchemaVersion:        "strategist-plugin-onboarding-plan/v1",
		RequiresConfirmation: true,
		Lock:                 lock,
		Inventory:            domain.PluginInventory{SchemaVersion: "strategist-plugin-inventory/v1", Instances: instances},
		Bindings:             bindings,
		Changes:              changes,
		RoleMigration:        roleMigration,
	}, nil
}

// logRoleBindingEvidence logs one line per role/provider binding evidence
// event (resolved, id_shadowing, role_binding_missing, role_binding_ambiguous)
// through the same slog-based, standalone-safe boundary applyWizardConfig
// already uses for its own install narration (tasks.md Task 6.1). This is a
// real, non-test call site — RoleBindingTelemetryEvent/Evidence() previously
// had none.
func logRoleBindingEvidence(events []telemetry.Event) {
	for _, ev := range events {
		attrs := make([]any, 0, len(ev.Attributes)*2+2)
		attrs = append(attrs, telemetry.AttrComponent, "install")
		for k, v := range ev.Attributes {
			attrs = append(attrs, k, v)
		}
		slog.Info(ev.Name, attrs...)
	}
}

func (p pluginOnboardingPlan) Preview() string {
	var b strings.Builder
	b.WriteString("plugin onboarding plan\n")
	b.WriteString("lock ")
	b.WriteString(p.Lock.GraphDigest)
	b.WriteString("\n")
	for _, change := range p.Changes {
		b.WriteString(change)
		b.WriteString("\n")
	}
	return b.String()
}

func applyPluginOnboardingPlan(store *lifecycle.Store, plan pluginOnboardingPlan, probe pluginProbeFunc) error {
	mergeInventory(store, plan.Inventory.Instances)
	for _, desired := range plan.Bindings {
		if err := applyPluginBinding(store, desired, probe); err != nil {
			return err
		}
	}
	return nil
}

func applyPluginBinding(store *lifecycle.Store, desired domain.SlotBinding, probe pluginProbeFunc) error {
	current, ok := store.Binding(desired.Slot)
	if !ok {
		store.Bindings = append(store.Bindings, domain.SlotBinding{
			SchemaVersion:       "strategist-plugin-binding/v1",
			Slot:                desired.Slot,
			InstalledInstanceID: desired.InstalledInstanceID,
			Generation:          0,
			Status:              desired.Status,
		})
		return nil
	}
	if current.InstalledInstanceID == desired.InstalledInstanceID {
		return nil
	}
	instance, ok := store.Instance(desired.InstalledInstanceID)
	if !ok {
		return fmt.Errorf("planned_instance_missing: %s", desired.InstalledInstanceID)
	}
	return switchPluginBinding(store, current, desired, instance, probe)
}

func switchPluginBinding(store *lifecycle.Store, current, desired domain.SlotBinding, instance domain.InstalledInstance, probe pluginProbeFunc) error {
	txID := "plugin-onboarding-" + desired.Slot + "-" + desired.InstalledInstanceID
	tx, err := store.Begin(txID, desired.Slot, desired.InstalledInstanceID)
	if err != nil {
		return fmt.Errorf("begin plugin lifecycle transaction: %w", err)
	}
	if err := store.Stage(tx.ID); err != nil {
		return fmt.Errorf("stage plugin lifecycle transaction: %w", err)
	}
	if err := store.Probe(tx.ID, probe(desired, instance)); err != nil {
		return fmt.Errorf("probe plugin lifecycle transaction: %w", err)
	}
	if err := store.Activate(tx.ID, current.Generation); err != nil {
		if rollbackErr := store.Rollback(tx.ID); rollbackErr != nil {
			return fmt.Errorf("activate plugin lifecycle transaction: %w; rollback: %v", err, rollbackErr)
		}
		return fmt.Errorf("activate plugin lifecycle transaction: %w", err)
	}
	return nil
}

func wizardSlots(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		"discovery":  wc.DiscoveryProvider,
		"refinement": wc.RefinementProvider,
		"execution":  wc.ExecutionProvider,
	}
}

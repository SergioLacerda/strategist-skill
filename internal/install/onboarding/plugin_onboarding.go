package onboarding

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/application/installplan"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domaininstallplan "github.com/SergioLacerda/strategist-skill/internal/domain/installplan"
	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// PluginOnboardingPlan is the resolved, previewable set of binding changes an install applies for the chosen slots.
type PluginOnboardingPlan struct {
	SchemaVersion        string
	RequiresConfirmation bool
	// InstallPlan is the shared, read-only ROSTER plan consumed by both the
	// interactive Wizard and headless installation before activation.
	InstallPlan domaininstallplan.InstallPlan
	Options     plugincatalog.InstallOptionSet
	Roster      domainroster.WeaponRosterArtifact
	Selections  []domainroster.WeaponSelectionArtifact
	Lock        domain.PluginLock
	Inventory   domain.PluginInventory
	Bindings    []domain.SlotBinding
	Changes     []string
	// RoleMigration is the Role/Provider convergence preview for the same
	// slots (tasks.md Task 4.1/4.2, strategist-papeis-personagens-skills-nativas).
	// It is additive evidence over the legacy Lock/Bindings above, never a
	// replacement for them (Decision 4: lifecycle reuse) — a role/provider
	// resolution failure for one slot never fails plugin onboarding as a
	// whole, since the legacy catalog/lock resolution above already is the
	// authoritative gate for whether a slot's provider is usable.
	RoleMigration RoleProviderMigrationPreview
	// CustomProviders records local-first packages resolved for explicitly
	// typed Custom providers. They are plan evidence, not embedded catalog
	// entries.
	CustomProviders map[string]customprovider.Resolution
}

type pluginProbeFunc func(domain.SlotBinding, domain.InstalledInstance) bool

type pluginProbeResultFunc func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome

// BindInstallContext binds the plan to the install mode, base path and chosen slots and modes.
func (p *PluginOnboardingPlan) BindInstallContext(mode, basePath string, slots, modes map[string]string) error {
	installPlan, err := installplan.PlanInstall(installplan.Input{
		Stage: domain.StageRoster, Mode: mode, BasePath: basePath,
		Slots: slots, SlotModes: modes, Lock: p.Lock, Bindings: p.Bindings,
	})
	if err != nil {
		return fmt.Errorf("create install plan: %w", err)
	}
	p.InstallPlan = installPlan
	return nil
}

// PlanPluginOnboarding resolves the onboarding plan for the chosen slots against catalog.
func PlanPluginOnboarding(extractor domain.FileExtractor, catalog plugincatalog.PluginCatalog, slots map[string]string) (PluginOnboardingPlan, error) {
	return PlanPluginOnboardingWithModes(extractor, catalog, slots, nil)
}

// LogRoleBindingEvidence logs one line per role/provider binding evidence
// event (resolved, id_shadowing, role_binding_missing, role_binding_ambiguous)
// through the same slog-based, standalone-safe boundary applyWizardConfig
// already uses for its own install narration (tasks.md Task 6.1). This is a
// real, non-test call site — RoleBindingTelemetryEvent/Evidence() previously
// had none.
func LogRoleBindingEvidence(events []telemetry.Event) {
	for _, ev := range events {
		attrs := make([]any, 0, len(ev.Attributes)*2+2)
		attrs = append(attrs, telemetry.AttrComponent, "install")
		for k, v := range ev.Attributes {
			attrs = append(attrs, k, v)
		}
		slog.Info(ev.Name, attrs...)
	}
}

// Preview returns the human-readable change preview of the plan.
func (p PluginOnboardingPlan) Preview() string {
	var b strings.Builder
	b.WriteString("plugin onboarding plan\n")
	b.WriteString("install plan ")
	b.WriteString(p.InstallPlan.PlanDigest)
	b.WriteString("\n")
	b.WriteString("lock ")
	b.WriteString(p.Lock.GraphDigest)
	b.WriteString("\n")
	for _, change := range p.Changes {
		b.WriteString(change)
		b.WriteString("\n")
	}
	return b.String()
}

// ApplyPluginOnboardingPlan applies the bindings of plan to store, probing each activated binding with probe.
func ApplyPluginOnboardingPlan(store *lifecycle.Store, plan PluginOnboardingPlan, probe pluginProbeFunc) error {
	mergeInventory(store, plan.Inventory.Instances)
	for _, desired := range plan.Bindings {
		if err := ApplyPluginBinding(store, desired, probe); err != nil {
			return err
		}
	}
	return nil
}

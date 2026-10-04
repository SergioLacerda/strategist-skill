package install

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/application/installplan"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domaininstallplan "github.com/SergioLacerda/strategist-skill/internal/domain/installplan"
	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

type pluginOnboardingPlan struct {
	SchemaVersion        string
	RequiresConfirmation bool
	// InstallPlan is the shared, read-only ROSTER plan consumed by both the
	// interactive Wizard and headless installation before activation.
	InstallPlan domaininstallplan.InstallPlan
	Options     installOptionSet
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
	CustomProviders map[string]customProviderResolution
}

type pluginProbeFunc func(domain.SlotBinding, domain.InstalledInstance) bool

type pluginProbeResultFunc func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome

func (p *pluginOnboardingPlan) bindInstallContext(mode, basePath string, slots, modes map[string]string) error {
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

func planPluginOnboarding(extractor domain.FileExtractor, catalog pluginCatalog, slots map[string]string) (pluginOnboardingPlan, error) {
	return planPluginOnboardingWithModes(extractor, catalog, slots, nil)
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

func applyPluginOnboardingPlan(store *lifecycle.Store, plan pluginOnboardingPlan, probe pluginProbeFunc) error {
	mergeInventory(store, plan.Inventory.Instances)
	for _, desired := range plan.Bindings {
		if err := applyPluginBinding(store, desired, probe); err != nil {
			return err
		}
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

func wizardSlotModes(wc domain.WizardConfig) map[string]string {
	return map[string]string{
		"discovery":  wc.DiscoveryMode,
		"refinement": wc.RefinementMode,
		"execution":  wc.ExecutionMode,
	}
}

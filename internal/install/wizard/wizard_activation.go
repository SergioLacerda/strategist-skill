package wizard

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/onboarding"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/rankedbinding"
)

func validateWizardPlanInputs(providerRisk map[string]string, wc domain.WizardConfig) error {
	// Task 6: pause on a custom skill the registry has no opinion on AND
	// that cannot be resolved as an already-installed workspace skill —
	// distinct from validateProvider's non-blocking risk-mismatch warning
	// already applied per-field earlier in runWizard.
	if err := customprovider.CheckCustomSkillAvailability(providerRisk, plugincatalog.WizardSlots(wc)); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}
	return nil
}

// ActivateWizardPlan validates the wizard selections and then activates plan, returning the resulting lock file.
func ActivateWizardPlan(extractor domain.FileExtractor, catalog plugincatalog.PluginCatalog, wc domain.WizardConfig, strategistDir string, plan onboarding.PluginOnboardingPlan, verbose bool) (domain.PluginLockFile, error) {
	if err := ValidateWizardActivationInputs(extractor, wc, plan, strategistDir, verbose); err != nil {
		return domain.PluginLockFile{}, err
	}
	lockFile, err := ApplyWizardActivation(extractor, catalog, wc, strategistDir, plan)
	if err != nil {
		return domain.PluginLockFile{}, err
	}
	return lockFile, nil
}

// ValidateWizardActivationInputs validates the install plan, the role/provider migration preview and the leveling
// policy of the wizard selections before anything is activated.
func ValidateWizardActivationInputs(extractor domain.FileExtractor, wc domain.WizardConfig, plan onboarding.PluginOnboardingPlan, strategistDir string, verbose bool) error {
	if err := plan.InstallPlan.Validate(); err != nil {
		return fmt.Errorf("wizard: invalid install plan: %w", err)
	}
	// Task 4.1: show Role separately from its resolved/candidate Providers
	// instead of only validating the legacy slot/catalog shape above.
	printMigrationPreview(plan.RoleMigration, verbose)
	onboarding.LogRoleBindingEvidence(plan.RoleMigration.Evidence())
	if err := ValidateWizardRoleBindings(plan.RoleMigration); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}
	if err := ValidateWizardLeveling(strategistDir, wc, extractor); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}
	return nil
}

// ApplyWizardActivation drives the resolved bindings through the staged/probed/active lifecycle and returns the
// persisted lock file.
func ApplyWizardActivation(_ domain.FileExtractor, catalog plugincatalog.PluginCatalog, wc domain.WizardConfig, strategistDir string, plan onboarding.PluginOnboardingPlan) (domain.PluginLockFile, error) {
	// .analysis/refined/20260913-embedded-skill-directory-catalog Task 5:
	// actually drive the resolved bindings through the real staged/probed/
	// active lifecycle instead of only previewing them —
	// ApplyRoleProviderMigration previously had zero production callers.
	lockFile, err := onboarding.ActivateRoleProviderMigration(strategistDir, plan.Lock, plan.RoleMigration)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: activate role/provider migration: %w", err)
	}
	if err := ValidatePersistedRoleBindings(lockFile, plan.RoleMigration); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	customprovider.AnnotateCustomProviderInstances(&lockFile, plan.CustomProviders)

	// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-005:
	// for every slot the Wizard resolved to Ranked, activate (copy) the
	// pre-generated certification-time SlotBinding over the Custom-mode one
	// activateRoleProviderMigration just wrote — every other slot (the
	// overwhelming majority: every existing installation) is untouched.
	lockFile, err = rankedbinding.ApplyRankedBindingChoices(catalog, wc, lockFile)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	lockFile, err = rankedbinding.EnrichLockBindingMetadata(catalog, lockFile)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: enrich Role/Weapon bindings: %w", err)
	}
	// Last step, so no later enrichment can overwrite the complete Custom
	// binding: normalize each typed Custom host Weapon into the installed
	// package representation and bind it with full evidence.
	lockFile, err = customprovider.MaterializeCustomProviders(strategistDir, lockFile, plan.CustomProviders)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	lockFile, err = rankedbinding.AttachWeaponBindingArtifacts(lockFile)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	return lockFile, nil
}

// printMigrationPreview shows the role/provider migration preview only in
// verbose mode; the binding evidence stays available through the INFO events.
func printMigrationPreview(preview onboarding.RoleProviderMigrationPreview, verbose bool) {
	if verbose {
		fmt.Println(preview.Preview())
	}
}

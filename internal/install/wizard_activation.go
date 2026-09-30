package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateWizardPlanInputs(providerRisk map[string]string, wc domain.WizardConfig) error {
	// Task 6: pause on a custom skill the registry has no opinion on AND
	// that cannot be resolved as an already-installed workspace skill —
	// distinct from validateProvider's non-blocking risk-mismatch warning
	// already applied per-field earlier in runWizard.
	if err := checkCustomSkillAvailability(providerRisk, wizardSlots(wc)); err != nil {
		return fmt.Errorf("wizard: %w", err)
	}
	return nil
}

func activateWizardPlan(extractor domain.FileExtractor, catalog pluginCatalog, wc domain.WizardConfig, strategistDir string, plan pluginOnboardingPlan) (domain.PluginLockFile, error) {
	// Task 4.1: show Role separately from its resolved/candidate Providers
	// instead of only validating the legacy slot/catalog shape above.
	fmt.Println(plan.RoleMigration.Preview())
	logRoleBindingEvidence(plan.RoleMigration.Evidence())
	if err := validateWizardRoleBindings(plan.RoleMigration); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	if err := validateWizardLeveling(strategistDir, wc, extractor); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}

	// .analysis/refined/20260913-embedded-skill-directory-catalog Task 5:
	// actually drive the resolved bindings through the real staged/probed/
	// active lifecycle instead of only previewing them —
	// ApplyRoleProviderMigration previously had zero production callers.
	lockFile, err := activateRoleProviderMigration(strategistDir, plan.Lock, plan.RoleMigration)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: activate role/provider migration: %w", err)
	}
	if err := validatePersistedRoleBindings(lockFile, plan.RoleMigration); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	annotateCustomProviderInstances(&lockFile, plan.CustomProviders)

	// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-005:
	// for every slot the Wizard resolved to Ranked, activate (copy) the
	// pre-generated certification-time SlotBinding over the Custom-mode one
	// activateRoleProviderMigration just wrote — every other slot (the
	// overwhelming majority: every existing installation) is untouched.
	lockFile, err = applyRankedBindingChoices(catalog, wc, lockFile)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}
	lockFile, err = enrichLockBindingMetadata(catalog, lockFile)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: enrich Role/Weapon bindings: %w", err)
	}
	return lockFile, nil
}

package install

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/onboarding"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/rankedbinding"
	"gopkg.in/yaml.v3"
)

// refreshInstalledRankedBindings repairs immutable Ranked records after an
// upgrade or reinstall that preserves active.yaml. It does not infer Ranked
// mode from a provider id, so a Custom binding remains the user's choice.
func (s Service) refreshInstalledRankedBindings(strategistDir string) error {
	lockFile, ok, err := readRankedRefreshLock(strategistDir)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	catalog, err := plugincatalog.LoadPluginCatalog(s.Extractor)
	if err != nil {
		return fmt.Errorf("load plugin catalog: %w", err)
	}
	refreshed, err := refreshRankedLock(catalog, lockFile)
	if err != nil {
		return err
	}
	if err := plugincatalog.WritePluginLockFile(strategistDir, refreshed); err != nil {
		return fmt.Errorf("write plugins.lock: %w", err)
	}
	return nil
}

func readRankedRefreshLock(strategistDir string) (domain.PluginLockFile, bool, error) {
	lockFile, err := plugincatalog.ReadPluginLockFile(strategistDir)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.PluginLockFile{}, false, nil
		}
		return domain.PluginLockFile{}, false, fmt.Errorf("read plugins.lock: %w", err)
	}
	return lockFile, hasPersistedRankedBinding(lockFile), nil
}

func refreshRankedLock(catalog plugincatalog.PluginCatalog, lockFile domain.PluginLockFile) (domain.PluginLockFile, error) {
	refreshed, err := rankedbinding.RefreshPersistedRankedBindings(catalog, lockFile)
	if err != nil {
		return domain.PluginLockFile{}, err //nolint:wrapcheck // sibling package of the same module; its error already names the binding
	}
	return rankedbinding.AttachWeaponBindingArtifacts(refreshed) //nolint:wrapcheck // sibling package of the same module; its error already names the binding
}

func hasPersistedRankedBinding(lockFile domain.PluginLockFile) bool {
	for _, binding := range lockFile.Bindings {
		if binding.EffectiveMode() == domain.SlotBindingModeRanked {
			return true
		}
	}
	return false
}

// activateSilentRoleProviderBindings resolves and persists plugins.lock for
// a silent install's template
// slots, mirroring applyWizardConfig's activation path for interactively
// chosen slots. Split out of installer_config.go to keep that file under the
// repo's file-size budget.
func (s Service) activateSilentRoleProviderBindings(strategistDir string, activeYAMLData []byte) error {
	slots, catalog, plan, err := s.loadSilentBindingPlan(activeYAMLData)
	if err != nil {
		return err
	}
	lockFile, err := onboarding.ActivateRoleProviderMigration(strategistDir, plan.Lock, plan.RoleMigration)
	if err != nil {
		return fmt.Errorf("activate role/provider migration: %w", err)
	}
	lockFile, err = rankedbinding.PromoteRuntimeProvidersToRanked(catalog, slots, lockFile)
	if err != nil {
		return fmt.Errorf("promote ranked runtime providers: %w", err)
	}
	lockFile, err = rankedbinding.EnrichLockBindingMetadata(catalog, lockFile)
	if err != nil {
		return fmt.Errorf("enrich Role/Weapon bindings: %w", err)
	}
	lockFile, err = rankedbinding.AttachWeaponBindingArtifacts(lockFile)
	if err != nil {
		return err //nolint:wrapcheck // sibling package of the same module; its error already names the binding
	}
	if err := persistSilentBindings(strategistDir, lockFile); err != nil {
		return err
	}
	return nil
}

func (s Service) loadSilentBindingPlan(activeYAMLData []byte) (map[string]string, plugincatalog.PluginCatalog, onboarding.PluginOnboardingPlan, error) {
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(activeYAMLData, &cfg); err != nil {
		return nil, plugincatalog.PluginCatalog{}, onboarding.PluginOnboardingPlan{}, fmt.Errorf("parse active.yaml template: %w", err)
	}
	slots := map[string]string{
		string(domain.SlotDiscovery):  cfg.Slots[string(domain.SlotDiscovery)],
		string(domain.SlotRefinement): cfg.Slots[string(domain.SlotRefinement)],
		string(domain.SlotExecution):  cfg.Slots[string(domain.SlotExecution)],
	}
	catalog, err := plugincatalog.LoadPluginCatalog(s.Extractor)
	if err != nil {
		return nil, plugincatalog.PluginCatalog{}, onboarding.PluginOnboardingPlan{}, fmt.Errorf("load plugin catalog: %w", err)
	}
	plan, err := onboarding.PlanPluginOnboarding(s.Extractor, catalog, slots)
	if err != nil {
		return nil, plugincatalog.PluginCatalog{}, onboarding.PluginOnboardingPlan{}, fmt.Errorf("plugin onboarding plan: %w", err)
	}
	if err := plan.BindInstallContext(cfg.Mode, cfg.BasePath, slots, nil); err != nil {
		return nil, plugincatalog.PluginCatalog{}, onboarding.PluginOnboardingPlan{}, fmt.Errorf("install plan context: %w", err)
	}
	return slots, catalog, plan, nil
}

func persistSilentBindings(strategistDir string, lockFile domain.PluginLockFile) error {
	if len(lockFile.Bindings) == 0 {
		return nil
	}
	if err := plugincatalog.WritePluginLockFile(strategistDir, lockFile); err != nil {
		return fmt.Errorf("write plugins.lock: %w", err)
	}
	return nil
}

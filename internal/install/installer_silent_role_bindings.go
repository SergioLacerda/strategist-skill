package install

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// refreshInstalledRankedBindings repairs immutable Ranked records after an
// upgrade or reinstall that preserves active.yaml. It does not infer Ranked
// mode from a provider id, so a Custom binding remains the user's choice.
func (s Service) refreshInstalledRankedBindings(strategistDir string) error {
	lockFile, err := readPluginLockFile(strategistDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read plugins.lock: %w", err)
	}
	if !hasPersistedRankedBinding(lockFile) {
		return nil
	}
	catalog, err := loadPluginCatalog(s.Extractor)
	if err != nil {
		return fmt.Errorf("load plugin catalog: %w", err)
	}
	refreshed, err := refreshPersistedRankedBindings(catalog, lockFile)
	if err != nil {
		return err
	}
	if err := writePluginLockFile(strategistDir, refreshed); err != nil {
		return fmt.Errorf("write plugins.lock: %w", err)
	}
	return nil
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
	lockFile, err := activateRoleProviderMigration(strategistDir, plan.Lock, plan.RoleMigration)
	if err != nil {
		return fmt.Errorf("activate role/provider migration: %w", err)
	}
	lockFile, err = promoteRuntimeProvidersToRanked(catalog, slots, lockFile)
	if err != nil {
		return fmt.Errorf("promote ranked runtime providers: %w", err)
	}
	lockFile, err = enrichLockBindingMetadata(catalog, lockFile)
	if err != nil {
		return fmt.Errorf("enrich Role/Weapon bindings: %w", err)
	}
	if err := persistSilentBindings(strategistDir, lockFile); err != nil {
		return err
	}
	return nil
}

func (s Service) loadSilentBindingPlan(activeYAMLData []byte) (map[string]string, pluginCatalog, pluginOnboardingPlan, error) {
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(activeYAMLData, &cfg); err != nil {
		return nil, pluginCatalog{}, pluginOnboardingPlan{}, fmt.Errorf("parse active.yaml template: %w", err)
	}
	slots := map[string]string{
		"discovery":  cfg.Slots["discovery"],
		"refinement": cfg.Slots["refinement"],
		"execution":  cfg.Slots["execution"],
	}
	catalog, err := loadPluginCatalog(s.Extractor)
	if err != nil {
		return nil, pluginCatalog{}, pluginOnboardingPlan{}, fmt.Errorf("load plugin catalog: %w", err)
	}
	plan, err := planPluginOnboarding(s.Extractor, catalog, slots)
	if err != nil {
		return nil, pluginCatalog{}, pluginOnboardingPlan{}, fmt.Errorf("plugin onboarding plan: %w", err)
	}
	return slots, catalog, plan, nil
}

func persistSilentBindings(strategistDir string, lockFile domain.PluginLockFile) error {
	if len(lockFile.Bindings) == 0 {
		return nil
	}
	if err := writePluginLockFile(strategistDir, lockFile); err != nil {
		return fmt.Errorf("write plugins.lock: %w", err)
	}
	return nil
}

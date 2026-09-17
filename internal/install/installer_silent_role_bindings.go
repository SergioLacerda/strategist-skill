package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// activateSilentRoleProviderBindings resolves and persists plugins.lock plus
// the selected providers' skill manifests for a silent install's template
// slots, mirroring applyWizardConfig's activation path for interactively
// chosen slots. Split out of installer_config.go to keep that file under the
// repo's file-size budget.
func (s Service) activateSilentRoleProviderBindings(strategistDir string, activeYAMLData []byte) error {
	var cfg domain.ActiveConfig
	if err := yaml.Unmarshal(activeYAMLData, &cfg); err != nil {
		return fmt.Errorf("parse active.yaml template: %w", err)
	}
	slots := map[string]string{
		"discovery":  cfg.Slots["discovery"],
		"refinement": cfg.Slots["refinement"],
		"execution":  cfg.Slots["execution"],
	}
	catalog, err := loadPluginCatalog(s.Extractor)
	if err != nil {
		return fmt.Errorf("load plugin catalog: %w", err)
	}
	plan, err := planPluginOnboarding(s.Extractor, catalog, slots)
	if err != nil {
		return fmt.Errorf("plugin onboarding plan: %w", err)
	}
	lockFile, err := activateRoleProviderMigration(strategistDir, plan.Lock, plan.RoleMigration)
	if err != nil {
		return fmt.Errorf("activate role/provider migration: %w", err)
	}
	if err := persistSilentBindings(strategistDir, lockFile); err != nil {
		return err
	}
	wc := domain.WizardConfig{DiscoveryProvider: slots["discovery"], RefinementProvider: slots["refinement"]}
	if err := s.writeSelectedProviderManifests(strategistDir, wc); err != nil {
		return fmt.Errorf("write provider manifests: %w", err)
	}
	return nil
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

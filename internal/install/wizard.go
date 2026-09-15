package install

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

var defaultLangOptions = []string{"en", "pt-BR"}
var defaultModeOptions = []string{"pragmatic", "epic"}

// installableDefaultProviders, knownProviderRisk, and loadKnownProviders live
// in wizard_fallback_providers.go, split out to keep this file under the
// repo's file-size budget.

// skillConfig holds values read from the embedded skill.yaml active_config section.
type skillConfig struct {
	LangOptions []string
	ModeOptions []string
}

// loadSkillConfig reads skill.yaml from the extractor and extracts the language and mode
// option lists declared in active_config. Falls back to hardcoded defaults on any error.
func loadSkillConfig(extractor domain.FileExtractor) skillConfig {
	data, err := extractor.ReadFile(skillYAMLName)
	if err != nil {
		return skillConfig{LangOptions: defaultLangOptions, ModeOptions: defaultModeOptions}
	}
	var doc struct {
		ActiveConfig struct {
			Language struct {
				Values []string `yaml:"values"`
			} `yaml:"language"`
			Mode struct {
				Values []string `yaml:"values"`
			} `yaml:"mode"`
		} `yaml:"active_config"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return skillConfig{LangOptions: defaultLangOptions, ModeOptions: defaultModeOptions}
	}
	cfg := skillConfig{LangOptions: defaultLangOptions, ModeOptions: defaultModeOptions}
	if len(doc.ActiveConfig.Language.Values) > 0 {
		cfg.LangOptions = doc.ActiveConfig.Language.Values
	}
	if len(doc.ActiveConfig.Mode.Values) > 0 {
		cfg.ModeOptions = doc.ActiveConfig.Mode.Values
	}
	return cfg
}

// validateProvider returns a non-empty warning if a slot plugin is unknown or its
// declared risk_score does not match the expected risk for the slot.
func validateProvider(registry map[string]string, provider, expectedRisk string) string {
	risk, ok := registry[provider]
	if !ok {
		return fmt.Sprintf("warning: slot plugin %q is not in the known plugin catalog; "+
			"ensure its skill.yaml declares risk_score: %s", provider, expectedRisk)
	}
	if risk != expectedRisk {
		return fmt.Sprintf("warning: slot plugin %q has risk_score %q but slot requires %q; "+
			"preflight will block at runtime", provider, risk, expectedRisk)
	}
	return ""
}

// runWizard collects install configuration through p. strategistDir is the
// target installation's .strategist directory — passed through to
// validateAndActivatePluginPlan so a resolved Role/Provider binding can be
// persisted to plugins.lock (docs/adr/0037-wizard-role-binding-persistence.md).
// An empty strategistDir skips that persistence step (see
// activateRoleProviderMigration).
func runWizard(ctx context.Context, p Prompter, extractor domain.FileExtractor, strategistDir string) (_ domain.WizardConfig, retErr error) {
	_, span := telemetry.Tracer().Start(ctx, "install.wizard")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	catalog, catalogErr := loadPluginCatalog(extractor)
	if catalogErr != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: %w — fix plugins/catalog.yaml (or the embedded default) before running the wizard; the wizard no longer falls back to hardcoded defaults silently (see docs/adr/0035-embedded-weapon-fallback-policy.md)", catalogErr)
	}

	providerRisk := loadKnownProviders(extractor)
	skillCfg := loadSkillConfig(extractor)
	uiLang, docLang, chatLang, codeLang, b, err := promptLanguages(p, skillCfg)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	mode, basePath, err := promptWorkspace(p, b, skillCfg)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	discovery, refinement, execution, err := promptSlots(p, b, catalog, providerRisk)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	chestPath, err := promptTreasureChest(p, b)
	if err != nil {
		return domain.WizardConfig{}, err
	}

	wc := domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		UILanguage:         uiLang,
		DocLanguage:        normLang(docLang),
		ChatLanguage:       normLang(chatLang),
		CodeLanguage:       normLang(codeLang),
		DiscoveryProvider:  discovery,
		RefinementProvider: refinement,
		ExecutionProvider:  execution,
		TreasureChestPath:  chestPath,
	}
	lockFile, err := validateAndActivatePluginPlan(extractor, catalog, providerRisk, wc, strategistDir)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	wc.ResolvedPluginLock = lockFile
	return wc, nil
}

// validateAndActivatePluginPlan runs every catalog-dependent Wizard check and
// activation step once a valid plugins/catalog.yaml has loaded: Task 6's
// custom-skill availability pause, the legacy plugin onboarding plan, the
// Role/Provider preview and evidence log, and Task 5's real staged-activation
// wiring. Split out of runWizard to keep runWizard's own branching shallow.
// The returned PluginLockFile is the resolved binding state to persist —
// zero-value when the migration was not fully resolved this run — the caller
// (applyWizardConfig) writes it to disk only after active.yaml lands
// (docs/adr/0037-wizard-role-binding-persistence.md).
func validateAndActivatePluginPlan(extractor domain.FileExtractor, catalog pluginCatalog, providerRisk map[string]string, wc domain.WizardConfig, strategistDir string) (domain.PluginLockFile, error) {
	// Task 6: pause on a custom skill the registry has no opinion on AND
	// that cannot be resolved as an already-installed workspace skill —
	// distinct from validateProvider's non-blocking risk-mismatch warning
	// already applied per-field earlier in runWizard.
	if err := checkCustomSkillAvailability(providerRisk, wizardSlots(wc)); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: %w", err)
	}

	plan, planErr := planPluginOnboarding(extractor, catalog, wizardSlots(wc))
	if planErr != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: plugin onboarding plan: %w", planErr)
	}
	// Task 4.1: show Role separately from its resolved/candidate Providers
	// instead of only validating the legacy slot/catalog shape above.
	fmt.Println(plan.RoleMigration.Preview())
	logRoleBindingEvidence(plan.RoleMigration.Evidence())
	if err := validateWizardRoleBindings(plan.RoleMigration); err != nil {
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
	return lockFile, nil
}

// validateWizardRoleBindings and validatePersistedRoleBindings live in
// wizard_role_validation.go, split out to keep this file under the repo's
// file-size budget.

// promptLanguages, selectLang, promptWorkspace, promptTreasureChest,
// promptSlots, and promptProvider live in wizard_prompts.go, split out to
// keep this file under the repo's file-size budget.

// normLang normalises language input to canonical form: "en" or "pt-BR".
// Accepts "pt" (skill.yaml canonical) and "pt-BR" (legacy/UI form).
func normLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") || strings.EqualFold(raw, "pt") {
		return "pt-BR"
	}
	return raw
}

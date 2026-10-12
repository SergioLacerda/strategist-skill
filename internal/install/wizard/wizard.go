package wizard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/install/customprovider"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/onboarding"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

// DefaultLangOptions are the documentation languages the wizard offers.
var DefaultLangOptions = []string{"en", "pt-BR"}

// DefaultModeOptions are the install modes the wizard offers.
var DefaultModeOptions = []string{"pragmatic", "epic"}

// installableDefaultProviders, knownProviderRisk, and loadKnownProviders live
// in wizard_fallback_providers.go, split out to keep this file under the
// repo's file-size budget.

// SkillConfig holds values read from the embedded skill.yaml active_config section.
type SkillConfig struct {
	LangOptions []string
	ModeOptions []string
}

// LoadSkillConfig reads skill.yaml from the extractor and extracts the language and mode
// option lists declared in active_config. Falls back to hardcoded defaults on any error.
func LoadSkillConfig(extractor domain.FileExtractor) SkillConfig {
	data, err := extractor.ReadFile(names.SkillYAMLName)
	if err != nil {
		return SkillConfig{LangOptions: DefaultLangOptions, ModeOptions: DefaultModeOptions}
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
		return SkillConfig{LangOptions: DefaultLangOptions, ModeOptions: DefaultModeOptions}
	}
	cfg := SkillConfig{LangOptions: DefaultLangOptions, ModeOptions: DefaultModeOptions}
	if len(doc.ActiveConfig.Language.Values) > 0 {
		cfg.LangOptions = doc.ActiveConfig.Language.Values
	}
	if len(doc.ActiveConfig.Mode.Values) > 0 {
		cfg.ModeOptions = doc.ActiveConfig.Mode.Values
	}
	return cfg
}

// ValidateProvider returns a non-empty warning if a slot plugin is unknown or its
// declared risk_score does not match the expected risk for the slot.
func ValidateProvider(registry map[string]string, provider, expectedRisk string) string {
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

// RunWizard collects install configuration through p. strategistDir is the
// target installation's .strategist directory — passed through to
// validateAndActivatePluginPlan so a resolved Role/Provider binding can be
// persisted to plugins.lock (docs/adr/0037-wizard-role-binding-persistence.md).
// An empty strategistDir skips that persistence step (see
// activateRoleProviderMigration).
func RunWizard(ctx context.Context, p Prompter, extractor domain.FileExtractor, strategistDir string, verbose bool) (_ domain.WizardConfig, retErr error) {
	_, span := telemetry.Tracer().Start(ctx, "install.wizard")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	catalog, catalogErr := plugincatalog.LoadPluginCatalog(extractor)
	if catalogErr != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: %w — fix plugins/catalog.yaml (or the embedded default) before running the wizard; the wizard no longer falls back to hardcoded defaults silently (see docs/adr/0035-embedded-weapon-fallback-policy.md)", catalogErr)
	}

	providerRisk := plugincatalog.LoadKnownProviders(extractor)
	workspaceRoot := ""
	if strategistDir != "" {
		workspaceRoot = filepath.Dir(strategistDir)
	}
	wc, err := collectWizardConfig(p, catalog, providerRisk, extractor, workspaceRoot, verbose, LoadWizardDefaults(strategistDir))
	if err != nil {
		return domain.WizardConfig{}, err
	}
	wc.IntegrationChoice, err = PromptIntegration(p, i18n.BundleFor(wc.UILanguage), strategistDir, os.Stdout)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	lockFile, err := ValidateAndActivatePluginPlan(extractor, catalog, providerRisk, wc, strategistDir, verbose)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	wc = customprovider.PinCustomInstanceRefs(wc, lockFile)
	wc.ResolvedPluginLock = lockFile
	return wc, nil
}

func collectWizardConfig(p Prompter, catalog plugincatalog.PluginCatalog, providerRisk map[string]string, extractor domain.FileExtractor, workspaceRoot string, verbose bool, defaults Defaults) (domain.WizardConfig, error) {
	skillCfg := LoadSkillConfig(extractor)
	uiLang, docLang, chatLang, codeLang, b, err := promptLanguages(p, skillCfg, defaults)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	mode, basePath, err := promptWorkspace(p, b, skillCfg, defaults)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	adrRoot, runbookRoot, err := PromptDocumentationRoots(p, b, basePath, workspaceRoot, defaults)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	discovery, refinement, execution, discoveryMode, refinementMode, executionMode, err := PromptSlots(p, b, catalog, providerRisk, verbose, defaults)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	chestPath, err := promptTreasureChest(p, b, verbose)
	if err != nil {
		return domain.WizardConfig{}, err
	}
	// LEVELING is an internal ability and defaults to automatic when the
	// optional active.yaml block is absent. Existing explicit manual/automatic
	// blocks remain readable by the runtime; the new-install wizard does not
	// expose that compatibility setting as an operator choice.
	return domain.WizardConfig{Mode: mode, BasePath: basePath, UILanguage: uiLang, DocLanguage: NormLang(docLang), ChatLanguage: NormLang(chatLang), CodeLanguage: NormLang(codeLang), DiscoveryProvider: discovery, RefinementProvider: refinement, ExecutionProvider: execution, DiscoveryMode: discoveryMode, RefinementMode: refinementMode, ExecutionMode: executionMode, TreasureChestPath: chestPath, AdrCanonicalPath: adrRoot, RunbookCanonicalPath: runbookRoot}, nil
}

// ValidateAndActivatePluginPlan runs every catalog-dependent Wizard check and
// activation step once a valid plugins/catalog.yaml has loaded: Task 6's
// custom-skill availability pause, the legacy plugin onboarding plan, the
// Role/Provider preview and evidence log, and Task 5's real staged-activation
// wiring. Split out of runWizard to keep runWizard's own branching shallow.
// The returned PluginLockFile is the resolved binding state to persist —
// zero-value when the migration was not fully resolved this run — the caller
// (applyWizardConfig) writes it to disk only after active.yaml lands
// (docs/adr/0037-wizard-role-binding-persistence.md).
func ValidateAndActivatePluginPlan(extractor domain.FileExtractor, catalog plugincatalog.PluginCatalog, providerRisk map[string]string, wc domain.WizardConfig, strategistDir string, verbose bool) (domain.PluginLockFile, error) {
	if err := validateWizardPlanInputs(providerRisk, wc); err != nil {
		return domain.PluginLockFile{}, err
	}
	plan, err := onboarding.PlanPluginOnboardingWithModes(extractor, catalog, plugincatalog.WizardSlots(wc), plugincatalog.WizardSlotModes(wc))
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: plugin onboarding plan: %w", err)
	}
	if err := plan.BindInstallContext(wc.Mode, wc.BasePath, plugincatalog.WizardSlots(wc), plugincatalog.WizardSlotModes(wc)); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("wizard: install plan context: %w", err)
	}
	return ActivateWizardPlan(extractor, catalog, wc, strategistDir, plan, verbose)
}

// validateWizardRoleBindings and validatePersistedRoleBindings live in
// wizard_role_validation.go, split out to keep this file under the repo's
// file-size budget.

// promptLanguages, selectLang, promptWorkspace, promptTreasureChest,
// promptSlots, and promptProvider live in wizard_prompts.go, split out to
// keep this file under the repo's file-size budget.

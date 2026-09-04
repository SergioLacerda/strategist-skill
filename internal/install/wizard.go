package install

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
	"gopkg.in/yaml.v3"
)

var defaultLangOptions = []string{"en", "pt-BR"}
var defaultModeOptions = []string{"pragmatic", "epic"}

// installableDefaultProviders lists providers that ship as an installable
// skill.yaml template. archivist is deliberately absent here: it is the
// native refinement role (roles/archivist.yaml), materialized like any other
// native role, not a skill package requiring its own install manifest.
var installableDefaultProviders = map[string]string{
	defaultDiscoveryProvider: "skills/brainstorming/skill.yaml",
	"openspec-explore":       "skills/openspec-explore/skill.yaml",
}

// knownProviderRisk is populated at wizard start from the embedded known-providers.yaml.
// The static map below is the fallback used only when the embed read fails.
var knownProviderRisk = map[string]string{
	defaultDiscoveryProvider:  "write_analysis",
	"openspec-explore":        "write_analysis",
	"openspec-propose":        "write_analysis",
	"openspec-apply-change":   "controlled",
	"openspec-archive-change": "write_analysis",
	nativeExecutionProvider:   "controlled",
	"sdd-ask":                 "controlled",
	"batata":                  "controlled",
	"sdd-diagnose":            "write_analysis",
	"sdd-converge":            "controlled",
	"sdd-correct":             "controlled",
	"sdd-stabilize":           "controlled",
	"sdd-validate-governance": "write_analysis",
	"sdd-organize":            "write_analysis",
	"sdd-review-architecture": "write_analysis",
	"archivist":               "write_analysis",
}

// loadKnownProviders reads templates/known-providers.yaml from the extractor and
// returns a provider→risk_score map. Falls back to the static map on any error.
func loadKnownProviders(extractor domain.FileExtractor) map[string]string {
	if catalog, err := loadPluginCatalog(extractor); err == nil {
		return catalogKnownProviderRisk(catalog)
	}
	data, err := extractor.ReadFile(knownProvidersTemplatePath)
	if err != nil {
		return knownProviderRisk
	}
	var doc struct {
		Providers map[string]string `yaml:"providers"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Providers) == 0 {
		return knownProviderRisk
	}
	return doc.Providers
}

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

// runWizard collects install configuration through p.
func runWizard(ctx context.Context, p Prompter, extractor domain.FileExtractor) (_ domain.WizardConfig, retErr error) {
	_, span := telemetry.Tracer().Start(ctx, "install.wizard")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

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
	discovery, refinement, execution, err := promptSlots(p, b, providerRisk)
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
	if catalog, catalogErr := loadPluginCatalog(extractor); catalogErr == nil {
		if _, planErr := planPluginOnboarding(catalog, wizardSlots(wc)); planErr != nil {
			return domain.WizardConfig{}, fmt.Errorf("wizard: plugin onboarding plan: %w", planErr)
		}
	}
	return wc, nil
}

func promptLanguages(p Prompter, skillCfg skillConfig) (uiLang, docLang, chatLang, codeLang string, b i18n.WizardStrings, err error) {
	uiLang, err = p.Select("Preferred language / Idioma preferido", "en", skillCfg.LangOptions)
	if err != nil {
		err = fmt.Errorf("wizard: ui_language: %w", err)
		return
	}
	uiLang = normLang(uiLang)
	b = i18n.BundleFor(uiLang)
	docLang, err = selectLang(p, b.PromptDocLang, skillCfg.LangOptions, "doc_language")
	if err != nil {
		return
	}
	chatLang, err = selectLang(p, b.PromptChatLang, skillCfg.LangOptions, "chat_language")
	if err != nil {
		return
	}
	codeLang, err = selectLang(p, b.PromptCodeLang, skillCfg.LangOptions, "code_language")
	return
}

func selectLang(p Prompter, prompt string, options []string, field string) (string, error) {
	value, err := p.Select(prompt, "en", options)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	return value, nil
}

func promptWorkspace(p Prompter, b i18n.WizardStrings, skillCfg skillConfig) (string, string, error) {
	mode, err := p.Select(b.PromptMode, "epic", skillCfg.ModeOptions)
	if err != nil {
		return "", "", fmt.Errorf("wizard: mode: %w", err)
	}
	basePath, err := p.Input(b.PromptBasePath, ".analysis")
	if err != nil {
		return "", "", fmt.Errorf("wizard: base_path: %w", err)
	}
	return mode, basePath, nil
}

func promptTreasureChest(p Prompter, b i18n.WizardStrings) (string, error) {
	fmt.Println(b.HeaderChest)
	chestPath, err := p.Input(b.PromptChestPath, "")
	if err != nil {
		return "", fmt.Errorf("wizard: treasure_chest: %w", err)
	}
	if chestPath == "" {
		fmt.Println(b.SkipChestHint)
	}
	return chestPath, nil
}

// promptSlots collects discovery and refinement slot providers. The execution slot is
// always the native `sniper` role — Strategist's built-in execution persona, not a
// governance/provider skill selectable from `.sdd/skills` (see mission
// 2026-07-25-wizard-execution-slot-native-sniper). The legacy execution prompt is still
// shown and consumed here so prompt count and scripted-input ordering stay stable, but
// its returned value is discarded: no typed input (e.g. `sdd-ask`) can ever leak into
// slots.execution.
func promptSlots(p Prompter, b i18n.WizardStrings, providerRisk map[string]string) (discovery, refinement, execution string, err error) {
	fmt.Println(b.HeaderSlots)
	discovery, err = promptProvider(p, b.PromptDiscovery, defaultDiscoveryProvider, []string{defaultDiscoveryProvider}, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", err
	}
	// openspec-explore is listed as a secondary, opt-in option: it requires a
	// separately installed skill and is no longer the recommended default (see
	// defaultRefinementProvider).
	refinement, err = promptProvider(p, b.PromptRefinement, defaultRefinementProvider, []string{defaultRefinementProvider, "openspec-explore"}, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", err
	}
	if _, err = promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution"); err != nil {
		return "", "", "", err
	}
	return discovery, refinement, nativeExecutionProvider, nil
}

func promptProvider(p Prompter, prompt, defaultVal string, options []string, customLabel string, providerRisk map[string]string, expectedRisk, field string) (string, error) {
	provider, err := p.SelectOrInput(prompt, defaultVal, options, customLabel)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	if w := validateProvider(providerRisk, provider, expectedRisk); w != "" {
		fmt.Println(w)
	}
	return provider, nil
}

// normLang normalises language input to canonical form: "en" or "pt-BR".
// Accepts "pt" (skill.yaml canonical) and "pt-BR" (legacy/UI form).
func normLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") || strings.EqualFold(raw, "pt") {
		return "pt-BR"
	}
	return raw
}

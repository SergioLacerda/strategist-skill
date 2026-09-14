package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

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
	discoveryDefault := defaultSkillBySlot["discovery"]
	discovery, err = promptProvider(p, b.PromptDiscovery, discoveryDefault, []string{discoveryDefault}, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", err
	}
	// openspec-propose is the refinement default (DEC-004); openspec-explore
	// remains listed as a secondary, opt-in option. archivist (the compatible
	// native-role fallback, roles/default.yaml) is intentionally not a manual
	// option here, matching discovery's own pattern where ranger is never a
	// manual option either — see defaultSkillBySlot.
	refinementDefault := defaultSkillBySlot["refinement"]
	refinement, err = promptProvider(p, b.PromptRefinement, refinementDefault, []string{refinementDefault, "openspec-explore"}, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
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

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

// Slot provider option-building (promptSlots, compatibleProviderOptions,
// excludedProviderOption, printExcludedCandidates) lives in
// wizard_prompts_slots.go; the Ranked-vs-Custom option/choice helpers
// (withRankedOption, splitRankedChoice, promptSlotProvider) live in
// wizard_prompts_ranked.go — both split out to keep this file under the
// repo's file-size budget.

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

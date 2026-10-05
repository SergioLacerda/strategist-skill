package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

func promptDocumentationRoots(p Prompter, b i18n.WizardStrings, basePath, workspaceRoot string) (string, string, error) {
	adrDefault := documentationRootDefault(basePath, workspaceRoot, "adr")
	runbookDefault := documentationRootDefault(basePath, workspaceRoot, "runbooks")
	adr, err := promptDocumentationRootWithin(p, b.PromptADRRoot, adrDefault, "adr.canonical_path", workspaceRoot)
	if err != nil {
		return "", "", err
	}
	runbooks, err := promptDocumentationRootWithin(p, b.PromptRunbookRoot, runbookDefault, "runbooks.canonical_path", workspaceRoot)
	if err != nil {
		return "", "", err
	}
	return adr, runbooks, nil
}

func promptDocumentationRoot(p Prompter, prompt, defaultPath, field string) (string, error) {
	return promptDocumentationRootWithin(p, prompt, defaultPath, field, "")
}

func promptDocumentationRootWithin(p Prompter, prompt, defaultPath, field, workspaceRoot string) (string, error) {
	value, err := p.Input(prompt, defaultPath)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	value, err = normalizeDocumentationRoot(value, field, workspaceRoot)
	if err != nil {
		return "", err
	}
	if err := domain.ValidateProjectRelativeDirectory(value, field); err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	return value, nil
}

func normalizeDocumentationRoot(value, field, workspaceRoot string) (string, error) {
	value = strings.TrimSpace(value)
	if !filepath.IsAbs(value) {
		return value, nil
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		return "", fmt.Errorf("wizard: %s: absolute path cannot be verified without a workspace root", field)
	}
	workspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: resolve workspace root: %w", field, err)
	}
	candidate, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: resolve documentation root: %w", field, err)
	}
	relative, err := filepath.Rel(filepath.Clean(workspace), filepath.Clean(candidate))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("wizard: %s must stay inside the selected project workspace", field)
	}
	return filepath.ToSlash(relative), nil
}

func documentationRootDefault(basePath, workspaceRoot, child string) string {
	root := filepath.Join(basePath, child)
	if !filepath.IsAbs(root) {
		return filepath.ToSlash(root)
	}
	if workspaceRoot != "" {
		if relative, err := filepath.Rel(workspaceRoot, root); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(relative)
		}
	}
	// Tests and callers without an installation target cannot resolve an absolute
	// workspace root. The base directory itself is the workspace in that case.
	return child
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
	fmt.Println(b.HeaderWorkspace)
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

func promptTreasureChest(p Prompter, b i18n.WizardStrings, verbose bool) (string, error) {
	if verbose {
		fmt.Println(b.HeaderChest)
	}
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

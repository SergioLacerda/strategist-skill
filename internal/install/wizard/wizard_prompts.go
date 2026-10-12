package wizard

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
)

// PromptDocumentationRoots asks for the ADR and runbook documentation roots.
func PromptDocumentationRoots(p Prompter, b i18n.WizardStrings, basePath, workspaceRoot string, defaults Defaults) (string, string, error) {
	adrDefault := OrDefault(defaults.ADRRoot, documentationRootDefault(basePath, workspaceRoot, "adr"), nil)
	runbookDefault := OrDefault(defaults.RunbookRoot, documentationRootDefault(basePath, workspaceRoot, "runbooks"), nil)
	adr, err := PromptDocumentationRootWithin(p, b.PromptADRRoot, adrDefault, names.AdrCanonicalPathKey, workspaceRoot)
	if err != nil {
		return "", "", err
	}
	runbooks, err := PromptDocumentationRootWithin(p, b.PromptRunbookRoot, runbookDefault, names.RunbooksCanonicalPathKey, workspaceRoot)
	if err != nil {
		return "", "", err
	}
	return adr, runbooks, nil
}

// PromptDocumentationRoot asks for one documentation root and validates it against the workspace.
func PromptDocumentationRoot(p Prompter, prompt, defaultPath, field string) (string, error) {
	return PromptDocumentationRootWithin(p, prompt, defaultPath, field, "")
}

// PromptDocumentationRootWithin asks for one documentation root that must stay inside workspaceRoot.
func PromptDocumentationRootWithin(p Prompter, prompt, defaultPath, field, workspaceRoot string) (string, error) {
	value, err := p.Input(prompt, defaultPath)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	value, err = NormalizeDocumentationRoot(value, field, workspaceRoot)
	if err != nil {
		return "", err
	}
	if err := domain.ValidateProjectRelativeDirectory(value, field); err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	return value, nil
}

// NormalizeDocumentationRoot returns value cleaned as a project-relative documentation directory.
func NormalizeDocumentationRoot(value, field, workspaceRoot string) (string, error) {
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
	if child == "runbooks" {
		// The Treasure Chest manifest and knowledge index both govern
		// docs/runbooks. Keep the installer default on that same source rather
		// than creating a second runtime root under base_path.
		return "docs/runbooks"
	}
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

func promptLanguages(p Prompter, skillCfg SkillConfig, defaults Defaults) (uiLang, docLang, chatLang, codeLang string, b i18n.WizardStrings, err error) {
	uiLang, err = p.Select("Preferred language / Idioma preferido", OrDefault(defaults.UI, "en", skillCfg.LangOptions), skillCfg.LangOptions)
	if err != nil {
		err = fmt.Errorf("wizard: ui_language: %w", err)
		return
	}
	uiLang = NormLang(uiLang)
	b = i18n.BundleFor(uiLang)
	docLang, err = selectLang(p, b.PromptDocLang, skillCfg.LangOptions, "doc_language", defaults.Docs)
	if err != nil {
		return
	}
	chatLang, err = selectLang(p, b.PromptChatLang, skillCfg.LangOptions, "chat_language", defaults.Chat)
	if err != nil {
		return
	}
	codeLang, err = selectLang(p, b.PromptCodeLang, skillCfg.LangOptions, "code_language", defaults.Code)
	return
}

func selectLang(p Prompter, prompt string, options []string, field, existing string) (string, error) {
	value, err := p.Select(prompt, OrDefault(existing, "en", options), options)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	return value, nil
}

func promptWorkspace(p Prompter, b i18n.WizardStrings, skillCfg SkillConfig, defaults Defaults) (string, string, error) {
	fmt.Println(b.HeaderWorkspace)
	mode, err := p.Select(b.PromptMode, OrDefault(defaults.Mode, "epic", skillCfg.ModeOptions), skillCfg.ModeOptions)
	if err != nil {
		return "", "", fmt.Errorf("wizard: mode: %w", err)
	}
	basePath, err := p.Input(b.PromptBasePath, OrDefault(defaults.BasePath, ".analysis", nil))
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
	if w := ValidateProvider(providerRisk, provider, expectedRisk); w != "" {
		fmt.Println(w)
	}
	return provider, nil
}

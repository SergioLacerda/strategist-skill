package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

var langOptions = []string{"en", "pt-BR"}

// runWizard collects install configuration through p.
func runWizard(p Prompter) (domain.WizardConfig, error) {
	// Prompt 1 — bilingual, bundle not yet chosen
	uiLang, err := p.Select("Preferred language / Idioma preferido", "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: ui_language: %w", err)
	}
	uiLang = normLang(uiLang)

	b := i18n.BundleFor(uiLang)

	docLang, err := p.Select(b.PromptDocLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: doc_language: %w", err)
	}
	docLang = normLang(docLang)

	chatLang, err := p.Select(b.PromptChatLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: chat_language: %w", err)
	}
	chatLang = normLang(chatLang)

	codeLang, err := p.Select(b.PromptCodeLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: code_language: %w", err)
	}
	codeLang = normLang(codeLang)

	mode, err := p.Select(b.PromptMode, "epic", []string{"pragmatic", "epic"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}

	basePath, err := p.Input(b.PromptBasePath, ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	adrRaw, err := p.Select(b.PromptAdr, "yes", []string{"yes", "no"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}
	adrEnabled := adrRaw == "yes"

	missionMode, err := p.Select(b.PromptMissionMode, "entrega_executada", []string{"analise", "entrega_revisada", "entrega_executada"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mission_mode: %w", err)
	}

	policy := domain.NewMissionPolicy(missionMode)
	doneScope := policy.DoneScope
	applyChanges := policy.ApplyChanges

	fmt.Println(b.HeaderSlots)

	discovery, err := p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, b.LabelCustomInput)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: execution: %w", err)
	}

	fmt.Println(b.HeaderChest)

	chestPath, err := p.Input(b.PromptChestPath, "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: treasure_chest: %w", err)
	}

	return domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		MissionMode:        missionMode,
		DoneScope:          doneScope,
		ApplyChanges:       applyChanges,
		UILanguage:         uiLang,
		DocLanguage:        docLang,
		ChatLanguage:       chatLang,
		CodeLanguage:       codeLang,
		AdrEnabled:         adrEnabled,
		DiscoveryProvider:  discovery,
		RefinementProvider: refinement,
		ExecutionProvider:  execution,
		TreasureChestPath:  chestPath,
	}, nil
}

// normLang normalises language input to canonical form: "en" or "pt-BR".
func normLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") {
		return "pt-BR"
	}
	return raw
}

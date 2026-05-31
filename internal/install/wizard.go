package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

type wizardStrings struct {
	PromptDocLang    string
	PromptChatLang   string
	PromptCodeLang   string
	PromptMode       string
	PromptBasePath   string
	PromptAdr        string
	HeaderSlots      string
	PromptDiscovery  string
	PromptRefinement string
	PromptExecution  string
	HeaderChest      string
	PromptChestPath  string
}

var bundleEN = wizardStrings{
	PromptDocLang:    "Documentation language",
	PromptChatLang:   "Chat/interaction language",
	PromptCodeLang:   "Code language",
	PromptMode:       "Mode",
	PromptBasePath:   "Base path for analysis workspace",
	PromptAdr:        "Enable ADR generation at mission end?",
	HeaderSlots:      "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:  "  Ranger / discovery provider",
	PromptRefinement: "  Arquivista / refinement provider",
	PromptExecution:  "  Sniper / execution provider",
	HeaderChest:      "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:  "  Knowledge source path (e.g. .sdd/source)",
}

var bundlePTBR = wizardStrings{
	PromptDocLang:    "Idioma da documentação",
	PromptChatLang:   "Idioma do chat/interação",
	PromptCodeLang:   "Idioma do código",
	PromptMode:       "Modo",
	PromptBasePath:   "Caminho base do workspace de análise",
	PromptAdr:        "Habilitar geração de ADR ao final da missão?",
	HeaderSlots:      "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:  "  Ranger / provedor de descoberta",
	PromptRefinement: "  Arquivista / provedor de refinamento",
	PromptExecution:  "  Sniper / provedor de execução",
	HeaderChest:      "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:  "  Caminho da base de conhecimento (ex: .sdd/source)",
}

func bundleFor(lang string) wizardStrings {
	if strings.EqualFold(lang, "pt-BR") || strings.EqualFold(lang, "pt-br") {
		return bundlePTBR
	}
	return bundleEN
}

var langOptions = []string{"en", "pt-BR"}

// runWizard collects install configuration through p.
func runWizard(p Prompter) (domain.WizardConfig, error) {
	// Prompt 1 — bilingual, bundle not yet chosen
	uiLang, err := p.Select("Preferred language / Idioma preferido", "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: ui_language: %w", err)
	}
	uiLang = normLang(uiLang)

	b := bundleFor(uiLang)

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

	mode, err := p.Select(b.PromptMode, "full", []string{"full", "lightweight", "minimal"})
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

	fmt.Println(b.HeaderSlots)

	customLabel := "(digitar outro...)"

	discovery, err := p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, customLabel)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, customLabel)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, customLabel)
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

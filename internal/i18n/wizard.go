package i18n

import "strings"

// WizardStrings holds all user-visible strings for the install wizard.
type WizardStrings struct {
	PromptDocLang     string
	PromptChatLang    string
	PromptCodeLang    string
	PromptMode        string
	PromptMissionMode string
	PromptBasePath    string
	PromptAdr         string
	HeaderSlots       string
	PromptDiscovery   string
	PromptRefinement  string
	PromptExecution   string
	HeaderChest       string
	PromptChestPath   string
	LabelCustomInput  string
}

// EN is the English wizard string bundle.
var EN = WizardStrings{
	PromptDocLang:     "Documentation language",
	PromptChatLang:    "Chat/interaction language",
	PromptCodeLang:    "Code language",
	PromptMode:        "Mode",
	PromptMissionMode: "DONE scope (analysis-only or analysis + implementation)\n  analise = analysis-only\n  entrega_revisada = analysis + handoff (no implementation)\n  entrega_executada = analysis + implementation",
	PromptBasePath:    "Base path for analysis workspace",
	PromptAdr:         "Enable ADR generation at mission end?",
	HeaderSlots:       "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:   "  Ranger / discovery provider",
	PromptRefinement:  "  Archivist / refinement provider",
	PromptExecution:   "  Sniper / execution provider",
	HeaderChest:       "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:   "  Knowledge source path (e.g. .sdd/source)",
	LabelCustomInput:  "(enter other...)",
}

// PT is the Portuguese wizard string bundle.
var PT = WizardStrings{
	PromptDocLang:     "Idioma da documentação",
	PromptChatLang:    "Idioma do chat/interação",
	PromptCodeLang:    "Idioma do código",
	PromptMode:        "Modo",
	PromptMissionMode: "Escopo do DONE (apenas análise ou análise + implementação)\n  analise = apenas análise\n  entrega_revisada = análise + handoff (sem implementação)\n  entrega_executada = análise + implementação",
	PromptBasePath:    "Caminho base do workspace de análise",
	PromptAdr:         "Habilitar geração de ADR ao final da missão?",
	HeaderSlots:       "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:   "  Ranger / provedor de descoberta",
	PromptRefinement:  "  Arquivista / provedor de refinamento",
	PromptExecution:   "  Sniper / provedor de execução",
	HeaderChest:       "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:   "  Caminho da base de conhecimento (ex: .sdd/source)",
	LabelCustomInput:  "(digitar outro...)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if strings.EqualFold(lang, "pt-BR") {
		return PT
	}
	return EN
}

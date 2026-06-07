package i18n

import "strings"

// WizardStrings holds all user-visible strings for the install wizard.
type WizardStrings struct {
	PromptDocLang       string
	PromptChatLang      string
	PromptCodeLang      string
	PromptMode          string
	PromptExecutionMode string
	PromptGitMode       string
	PromptBasePath      string
	PromptAdr           string
	HeaderSlots         string
	PromptDiscovery     string
	PromptRefinement    string
	PromptExecution     string
	HeaderChest         string
	PromptChestPath     string
	LabelCustomInput    string
}

// EN is the English wizard string bundle.
var EN = WizardStrings{
	PromptDocLang:       "Documentation language",
	PromptChatLang:      "Chat/interaction language",
	PromptCodeLang:      "Code language",
	PromptMode:          "Mode",
	PromptExecutionMode: "Execution mode\n  plan_only = analysis and refinement only\n  apply_workspace = analysis plus workspace changes",
	PromptGitMode:       "Git persistence mode\n  forbidden = no mutating Git commands\n  explicit_commit = mutating Git commands require per-command user approval",
	PromptBasePath:      "Base path for analysis workspace",
	PromptAdr:           "Enable ADR generation at mission end?",
	HeaderSlots:         "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:     "  Ranger / discovery provider",
	PromptRefinement:    "  Archivist / refinement provider",
	PromptExecution:     "  Sniper / execution provider",
	HeaderChest:         "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:     "  Knowledge source path (e.g. .sdd/source)",
	LabelCustomInput:    "(enter other...)",
}

// PT is the Portuguese wizard string bundle.
var PT = WizardStrings{
	PromptDocLang:       "Idioma da documentação",
	PromptChatLang:      "Idioma do chat/interação",
	PromptCodeLang:      "Idioma do código",
	PromptMode:          "Modo",
	PromptExecutionMode: "Modo de execução\n  plan_only = apenas análise e refinamento\n  apply_workspace = análise com alterações no workspace",
	PromptGitMode:       "Modo de persistência Git\n  forbidden = sem comandos Git mutáveis\n  explicit_commit = comandos Git mutáveis só com aprovação explícita por comando", //nolint:misspell
	PromptBasePath:      "Caminho base do workspace de análise",
	PromptAdr:           "Habilitar geração de ADR ao final da missão?",
	HeaderSlots:         "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:     "  Ranger / provedor de descoberta",
	PromptRefinement:    "  Arquivista / provedor de refinamento",
	PromptExecution:     "  Sniper / provedor de execução",
	HeaderChest:         "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:     "  Caminho da base de conhecimento (ex: .sdd/source)",
	LabelCustomInput:    "(digitar outro...)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if strings.EqualFold(lang, "pt-BR") {
		return PT
	}
	return EN
}

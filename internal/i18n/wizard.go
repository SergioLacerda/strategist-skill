package i18n

// WizardStrings holds all user-visible strings for the install wizard.
type WizardStrings struct {
	PromptDocLang    string
	PromptChatLang   string
	PromptCodeLang   string
	PromptMode       string
	PromptBasePath   string
	HeaderSlots      string
	PromptDiscovery  string
	PromptRefinement string
	PromptExecution  string
	HeaderChest      string
	PromptChestPath  string
	SkipChestHint    string
	LabelCustomInput string
}

// EN is the English wizard string bundle.
var EN = WizardStrings{
	PromptDocLang:    "Documentation language",
	PromptChatLang:   "Chat/interaction language",
	PromptCodeLang:   "Code language",
	PromptMode:       "Mode",
	PromptBasePath:   "Base path for analysis workspace",
	HeaderSlots:      "\nSlot plugins - which skill fills each mission role:",
	PromptDiscovery:  "  Ranger / discovery plugin",
	PromptRefinement: "  Archivist / refinement plugin",
	PromptExecution:  "  Sniper / documentation materialization plugin",
	HeaderChest:      "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:  "  Knowledge source path (e.g. .sdd/source)",
	SkipChestHint:    "  (skipped — edit .strategist/knowledge.index.yaml later to add sources)",
	LabelCustomInput: "(enter other...)",
}

// PT is the Portuguese wizard string bundle.
var PT = WizardStrings{
	PromptDocLang:    "Idioma da documentação",
	PromptChatLang:   "Idioma do chat/interação",
	PromptCodeLang:   "Idioma do código",
	PromptMode:       "Modo",
	PromptBasePath:   "Caminho base do workspace de análise",
	HeaderSlots:      "\nPlugins de slot - qual skill preenche cada papel da missão:",
	PromptDiscovery:  "  Ranger / plugin de descoberta",
	PromptRefinement: "  Arquivista / plugin de refinamento",
	PromptExecution:  "  Sniper / plugin de materialização de documentação",
	HeaderChest:      "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:  "  Caminho da base de conhecimento (ex: .sdd/source)",
	SkipChestHint:    "  (ignorado — edite .strategist/knowledge.index.yaml para adicionar fontes depois)",
	LabelCustomInput: "(digitar outro...)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if bundle, ok := wizardBundleFor(lang); ok {
		return bundle
	}
	return EN
}

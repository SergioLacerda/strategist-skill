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
	// LEVELING step: manual (host passthrough) or automatic (policy by
	// estimated load); one notice explains the selected mode.
	HeaderLeveling          string
	PromptLevelingMode      string
	LabelLevelingAutomatic  string
	LabelLevelingManual     string
	NoticeLevelingManual    string
	NoticeLevelingAutomatic string
	// NoteRankedHostNode is printed before a slot prompt whose Ranked option
	// runs the embedded OpenSpec bundle; %[1]s is the provider id and %[2]s the
	// minimum host Node version.
	NoteRankedHostNode string
}

// EN is the English wizard string bundle.
var EN = WizardStrings{
	PromptDocLang:           "Documentation language",
	PromptChatLang:          "Chat/interaction language",
	PromptCodeLang:          "Code language",
	PromptMode:              "Mode",
	PromptBasePath:          "Base path for analysis workspace",
	HeaderSlots:             "\nSlot plugins - which skill fills each mission role:",
	PromptDiscovery:         "  Ranger / discovery plugin",
	PromptRefinement:        "  Archivist / refinement plugin",
	PromptExecution:         "  Sniper / documentation materialization plugin",
	HeaderChest:             "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:         "  Knowledge source path (e.g. .sdd/source)",
	SkipChestHint:           "  (skipped — edit .strategist/knowledge.index.yaml later to add sources)",
	LabelCustomInput:        "(enter other...)",
	HeaderLeveling:          "\nModel x effort per role (LEVELING):",
	PromptLevelingMode:      "  Decide model and effort manually (default) or automatically (LEVELING policy)?",
	LabelLevelingAutomatic:  "automatic",
	LabelLevelingManual:     "manual",
	NoticeLevelingManual:    "  Manual: each role runs with the model and effort you set in your host/prompt. The Strategist flow never changes them.",
	NoticeLevelingAutomatic: "  Automatic: model and effort vary per role according to each role's estimated load (LEVELING policy).",
	NoteRankedHostNode:      "  Ranked %[1]s runs the embedded OpenSpec bundle with this machine's Node.js; Node.js >=%[2]s must be installed (OpenSpec and npm are not needed)",
}

// PT is the Portuguese wizard string bundle.
var PT = WizardStrings{
	PromptDocLang:           "Idioma da documentação",
	PromptChatLang:          "Idioma do chat/interação",
	PromptCodeLang:          "Idioma do código",
	PromptMode:              "Modo",
	PromptBasePath:          "Caminho base do workspace de análise",
	HeaderSlots:             "\nPlugins de slot - qual skill preenche cada papel da missão:",
	PromptDiscovery:         "  Ranger / plugin de descoberta",
	PromptRefinement:        "  Arquivista / plugin de refinamento",
	PromptExecution:         "  Sniper / plugin de materialização de documentação",
	HeaderChest:             "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:         "  Caminho da base de conhecimento (ex: .sdd/source)",
	SkipChestHint:           "  (ignorado — edite .strategist/knowledge.index.yaml para adicionar fontes depois)",
	LabelCustomInput:        "(digitar outro...)",
	HeaderLeveling:          "\nModelo x esforço por papel (LEVELING):",
	PromptLevelingMode:      "  Definir modelo e esforço manualmente (padrão) ou automaticamente (política LEVELING)?",
	LabelLevelingAutomatic:  "automático",
	LabelLevelingManual:     "manual",
	NoticeLevelingManual:    "  Manual: cada papel usa o modelo e o esforço que você definir no host/prompt. O fluxo do Strategist não altera esses valores.",
	NoticeLevelingAutomatic: "  Automático: modelo e esforço variam por papel conforme a carga estimada de cada um (política LEVELING).",
	NoteRankedHostNode:      "  %[1]s rankeado executa o bundle OpenSpec embutido com o Node.js desta máquina; é preciso ter Node.js >=%[2]s instalado (OpenSpec e npm não são necessários)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if bundle, ok := wizardBundleFor(lang); ok {
		return bundle
	}
	return EN
}

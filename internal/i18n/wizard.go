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
	// LEVELING step: automatic (policy) or manual model x effort per role.
	// PromptLevelingModel/Effort and SummaryLeveling take the role name (%[1]s);
	// SummaryLeveling also takes the resulting label (%[2]s).
	HeaderLeveling         string
	PromptLevelingMode     string
	LabelLevelingAutomatic string
	LabelLevelingManual    string
	PromptLevelingScope    string
	LabelLevelingAll       string
	LabelLevelingPerRole   string
	PromptLevelingModel    string
	PromptLevelingEffort   string
	SummaryLeveling        string
	// NoteRankedHostNode is printed before a slot prompt whose Ranked option
	// runs the embedded OpenSpec bundle; %[1]s is the provider id and %[2]s the
	// minimum host Node version.
	NoteRankedHostNode string
}

// EN is the English wizard string bundle.
var EN = WizardStrings{
	PromptDocLang:          "Documentation language",
	PromptChatLang:         "Chat/interaction language",
	PromptCodeLang:         "Code language",
	PromptMode:             "Mode",
	PromptBasePath:         "Base path for analysis workspace",
	HeaderSlots:            "\nSlot plugins - which skill fills each mission role:",
	PromptDiscovery:        "  Ranger / discovery plugin",
	PromptRefinement:       "  Archivist / refinement plugin",
	PromptExecution:        "  Sniper / documentation materialization plugin",
	HeaderChest:            "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:        "  Knowledge source path (e.g. .sdd/source)",
	SkipChestHint:          "  (skipped — edit .strategist/knowledge.index.yaml later to add sources)",
	LabelCustomInput:       "(enter other...)",
	HeaderLeveling:         "\nModel x effort per role (LEVELING):",
	PromptLevelingMode:     "  Decide model and effort manually (default) or automatically (LEVELING policy)?",
	LabelLevelingAutomatic: "automatic",
	LabelLevelingManual:    "manual",
	PromptLevelingScope:    "  Manual: one choice for all roles, or set each role?",
	LabelLevelingAll:       "all",
	LabelLevelingPerRole:   "per role",
	PromptLevelingModel:    "  Model for %[1]s (e.g. Sonnet; blank = decided automatically)",
	PromptLevelingEffort:   "  Effort for %[1]s",
	SummaryLeveling:        "  %[1]s -> %[2]s",
	NoteRankedHostNode:     "  Ranked %[1]s runs the embedded OpenSpec bundle with this machine's Node.js; Node.js >=%[2]s must be installed (OpenSpec and npm are not needed)",
}

// PT is the Portuguese wizard string bundle.
var PT = WizardStrings{
	PromptDocLang:          "Idioma da documentação",
	PromptChatLang:         "Idioma do chat/interação",
	PromptCodeLang:         "Idioma do código",
	PromptMode:             "Modo",
	PromptBasePath:         "Caminho base do workspace de análise",
	HeaderSlots:            "\nPlugins de slot - qual skill preenche cada papel da missão:",
	PromptDiscovery:        "  Ranger / plugin de descoberta",
	PromptRefinement:       "  Arquivista / plugin de refinamento",
	PromptExecution:        "  Sniper / plugin de materialização de documentação",
	HeaderChest:            "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:        "  Caminho da base de conhecimento (ex: .sdd/source)",
	SkipChestHint:          "  (ignorado — edite .strategist/knowledge.index.yaml para adicionar fontes depois)",
	LabelCustomInput:       "(digitar outro...)",
	HeaderLeveling:         "\nModelo x esforço por papel (LEVELING):",
	PromptLevelingMode:     "  Definir modelo e esforço manualmente (padrão) ou automaticamente (política LEVELING)?",
	LabelLevelingAutomatic: "automático",
	LabelLevelingManual:    "manual",
	PromptLevelingScope:    "  Manual: uma escolha para todos os papéis ou definir cada papel?",
	LabelLevelingAll:       "todos",
	LabelLevelingPerRole:   "por papel",
	PromptLevelingModel:    "  Modelo para %[1]s (ex: Sonnet; vazio = decidido automaticamente)",
	PromptLevelingEffort:   "  Esforço para %[1]s",
	SummaryLeveling:        "  %[1]s -> %[2]s",
	NoteRankedHostNode:     "  %[1]s rankeado executa o bundle OpenSpec embutido com o Node.js desta máquina; é preciso ter Node.js >=%[2]s instalado (OpenSpec e npm não são necessários)",
}

// BundleFor returns the WizardStrings for the given language code.
// Defaults to EN for unrecognised codes.
func BundleFor(lang string) WizardStrings {
	if bundle, ok := wizardBundleFor(lang); ok {
		return bundle
	}
	return EN
}

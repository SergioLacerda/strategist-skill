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
	// HeaderIntegration, NoteIntegrationData (%[1]s endpoint, %[2]s data
	// categories), PromptIntegration and NoteIntegrationPending belong to the
	// optional external provider question.
	HeaderIntegration      string
	NoteIntegrationData    string
	PromptIntegration      string
	NoteIntegrationPending string
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
	PromptChestPath:        "  Knowledge source path (e.g. governance/source)",
	SkipChestHint:          "  (skipped — edit .strategist/knowledge.index.yaml later to add sources)",
	LabelCustomInput:       "(enter other...)",
	HeaderIntegration:      "\nOptional integration — an external provider (JEV) that pre-checks handoffs:",
	NoteIntegrationData:    "  Endpoint: %[1]s\n  Data it may receive from a handoff: %[2]s\n  It only suggests: handoffs always keep working without it, and a failure or low confidence uses the normal path. The credential stays in your .env and is never stored.",
	PromptIntegration:      "  Use the external provider for handoff pre-checks",
	NoteIntegrationPending: "  The credential does not resolve yet (.env TYPESAFE_API_KEY): the choice is saved as pending and the normal path is used until it does.",
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
	PromptChestPath:        "  Caminho da base de conhecimento (ex: governance/source)",
	SkipChestHint:          "  (ignorado — edite .strategist/knowledge.index.yaml para adicionar fontes depois)",
	LabelCustomInput:       "(digitar outro...)",
	HeaderIntegration:      "\nIntegração opcional — um provedor externo (JEV) que pré-verifica os handoffs:",
	NoteIntegrationData:    "  Endpoint: %[1]s\n  Dados que ele pode receber de um handoff: %[2]s\n  Ele apenas sugere: os handoffs sempre funcionam sem ele, e uma falha ou baixa confiança usa o caminho normal. A credencial fica no seu .env e nunca é armazenada.",
	PromptIntegration:      "  Usar o provedor externo na pré-verificação de handoffs",
	NoteIntegrationPending: "  A credencial ainda não resolve (.env TYPESAFE_API_KEY): a escolha é salva como pendente e o caminho normal é usado até que resolva.",
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

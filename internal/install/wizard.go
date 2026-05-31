package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	ErrInvalidVal    string
}

var bundleEN = wizardStrings{
	PromptDocLang:    "Documentation language (en/pt-BR) [en]: ",
	PromptChatLang:   "Chat/interaction language (en/pt-BR) [en]: ",
	PromptCodeLang:   "Code language (en/pt-BR) [en]: ",
	PromptMode:       "Mode (full/lightweight/minimal) [full]: ",
	PromptBasePath:   "Base path for analysis workspace [.analysis]: ",
	PromptAdr:        "Enable ADR generation at mission end? (yes/no) [yes]: ",
	HeaderSlots:      "\nSlot providers — which skill fills each mission role:",
	PromptDiscovery:  "  Ranger / discovery provider [brainstorming]: ",
	PromptRefinement: "  Arquivista / refinement provider [openspec-explore]: ",
	PromptExecution:  "  Sniper / execution provider [sdd-ask]: ",
	HeaderChest:      "\nTreasure chest — optional offline knowledge source for all slots:",
	PromptChestPath:  "  Knowledge source path (e.g. .sdd/source) [leave empty to skip]: ",
	ErrInvalidVal:    "  Invalid value %q. Accepted: %s\n",
}

var bundlePTBR = wizardStrings{
	PromptDocLang:    "Idioma da documentação (en/pt-BR) [en]: ",
	PromptChatLang:   "Idioma do chat/interação (en/pt-BR) [en]: ",
	PromptCodeLang:   "Idioma do código (en/pt-BR) [en]: ",
	PromptMode:       "Modo (full/lightweight/minimal) [full]: ",
	PromptBasePath:   "Caminho base do workspace de análise [.analysis]: ",
	PromptAdr:        "Habilitar geração de ADR ao final da missão? (yes/no) [yes]: ",
	HeaderSlots:      "\nProvedores de slot — qual skill preenche cada papel da missão:",
	PromptDiscovery:  "  Ranger / provedor de descoberta [brainstorming]: ",
	PromptRefinement: "  Arquivista / provedor de refinamento [openspec-explore]: ",
	PromptExecution:  "  Sniper / provedor de execução [sdd-ask]: ",
	HeaderChest:      "\nBaú do tesouro — base de conhecimento offline opcional para todos os slots:",
	PromptChestPath:  "  Caminho da base de conhecimento (ex: .sdd/source) [deixar vazio para pular]: ",
	ErrInvalidVal:    "  Valor inválido %q. Aceitos: %s\n",
}

func bundleFor(lang string) wizardStrings {
	if strings.EqualFold(lang, "pt-BR") || strings.EqualFold(lang, "pt-br") {
		return bundlePTBR
	}
	return bundleEN
}

// RunWizard prompts the user interactively for install configuration.
func RunWizard() (domain.WizardConfig, error) {
	return runWizard(os.Stdin)
}

// runWizard is the testable core of RunWizard, reading from r instead of os.Stdin.
func runWizard(r io.Reader) (domain.WizardConfig, error) {
	br := bufio.NewReader(r)

	// Prompt 1 — bilingual, bundle not yet chosen
	uiLangRaw, err := promptValidated(br, "Preferred language / Idioma preferido (en/pt-BR) [en]: ", "en", []string{"en", "pt-BR", "pt-br"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: ui_language: %w", err)
	}
	uiLang := normLang(uiLangRaw)

	b := bundleFor(uiLang)

	// Language dimensions
	docLangRaw, err := promptValidated(br, b.PromptDocLang, "en", []string{"en", "pt-BR", "pt-br"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: doc_language: %w", err)
	}
	docLang := normLang(docLangRaw)

	chatLangRaw, err := promptValidated(br, b.PromptChatLang, "en", []string{"en", "pt-BR", "pt-br"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: chat_language: %w", err)
	}
	chatLang := normLang(chatLangRaw)

	codeLangRaw, err := promptValidated(br, b.PromptCodeLang, "en", []string{"en", "pt-BR", "pt-br"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: code_language: %w", err)
	}
	codeLang := normLang(codeLangRaw)

	// Operational config
	mode, err := prompt(br, b.PromptMode, "full")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}

	basePath, err := prompt(br, b.PromptBasePath, ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	adrRaw, err := promptValidated(br, b.PromptAdr, "yes", []string{"yes", "no", "y", "n"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}
	adrEnabled := adrRaw == "yes" || adrRaw == "y"

	fmt.Println(b.HeaderSlots)

	discovery, err := prompt(br, b.PromptDiscovery, "brainstorming")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := prompt(br, b.PromptRefinement, "openspec-explore")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := prompt(br, b.PromptExecution, "sdd-ask")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: execution: %w", err)
	}

	fmt.Println(b.HeaderChest)

	chestPath, err := prompt(br, b.PromptChestPath, "")
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
// promptValidated already lowercases the accepted value, so "pt-br" arrives here lowercased.
func normLang(raw string) string {
	if raw == "pt-br" {
		return "pt-BR"
	}
	return raw
}

// prompt writes question to stdout and reads a trimmed line from r.
// Returns defaultVal if the user submits an empty line.
func prompt(r *bufio.Reader, question, defaultVal string) (string, error) {
	fmt.Print(question)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// promptValidated prompts and retries until the user provides one of the accepted values
// or submits an empty line (returning defaultVal).
func promptValidated(r *bufio.Reader, question, defaultVal string, accepted []string) (string, error) {
	for {
		val, err := prompt(r, question, defaultVal)
		if err != nil {
			return "", err
		}
		for _, a := range accepted {
			if strings.EqualFold(val, a) {
				return strings.ToLower(val), nil
			}
		}
		fmt.Printf("  Invalid value %q. Accepted: %s\n", val, strings.Join(accepted, ", "))
	}
}

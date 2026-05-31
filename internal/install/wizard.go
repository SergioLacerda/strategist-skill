package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// wizardStrings holds all user-facing prompt text for a single language.
type wizardStrings struct {
	langPrompt       string
	modePrompt       string
	basePathPrompt   string
	languagePrompt   string
	adrPrompt        string
	slotsHeader      string
	discoveryPrompt  string
	refinementPrompt string
	executionPrompt  string
	chestHeader      string
	chestPrompt      string
}

var stringsEN = wizardStrings{
	langPrompt:       "Language / Idioma? (en / pt-br) [en]: ",
	modePrompt:       "Mode (full/lightweight/minimal) [full]: ",
	basePathPrompt:   "Analysis workspace path [.analysis]: ",
	languagePrompt:   "Artifact language (pt/en) [pt]: ",
	adrPrompt:        "Enable ADR generation at mission end? (yes/no) [yes]: ",
	slotsHeader:      "\nSlot providers — which skill fills each mission role:",
	discoveryPrompt:  "  Ranger / discovery provider [brainstorming]: ",
	refinementPrompt: "  Archivist / refinement provider [openspec-explore]: ",
	executionPrompt:  "  Sniper / execution provider [sdd-ask]: ",
	chestHeader:      "\nTreasure chest — optional offline knowledge source for all slots:",
	chestPrompt:      "  Knowledge source path (e.g. docs/knowledge) [leave empty to skip]: ",
}

var stringsPTBR = wizardStrings{
	langPrompt:       "Language / Idioma? (en / pt-br) [en]: ",
	modePrompt:       "Modo (full/lightweight/minimal) [full]: ",
	basePathPrompt:   "Caminho do workspace de análise [.analysis]: ",
	languagePrompt:   "Idioma dos artefatos (pt/en) [pt]: ",
	adrPrompt:        "Habilitar geração de ADR ao final da missão? (yes/no) [yes]: ",
	slotsHeader:      "\nSlot providers — qual skill preenche cada papel da missão:",
	discoveryPrompt:  "  Ranger / provider de descoberta [brainstorming]: ",
	refinementPrompt: "  Arquivista / provider de refinamento [openspec-explore]: ",
	executionPrompt:  "  Sniper / provider de execução [sdd-ask]: ",
	chestHeader:      "\nTreasure chest — base de conhecimento offline opcional para todos os slots:",
	chestPrompt:      "  Caminho da base de conhecimento (ex. docs/knowledge) [deixar vazio para pular]: ",
}

// RunWizard prompts the user interactively for install configuration.
func RunWizard() (domain.WizardConfig, error) {
	return runWizard(os.Stdin)
}

// runWizard is the testable core of RunWizard, reading from r instead of os.Stdin.
func runWizard(r io.Reader) (domain.WizardConfig, error) {
	br := bufio.NewReader(r)

	// Language selection — bilingual prompt regardless of choice
	langRaw, err := promptValidated(br, stringsEN.langPrompt, "en", []string{"en", "pt-br", "pt"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: language_ui: %w", err)
	}
	s := stringsEN
	if langRaw == "pt-br" || langRaw == "pt" {
		s = stringsPTBR
	}

	mode, err := prompt(br, s.modePrompt, "full")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}

	basePath, err := prompt(br, s.basePathPrompt, ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	language, err := promptValidated(br, s.languagePrompt, "pt", []string{"pt", "en"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: language: %w", err)
	}

	adrRaw, err := promptValidated(br, s.adrPrompt, "yes", []string{"yes", "no", "y", "n"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}
	adrEnabled := adrRaw == "yes" || adrRaw == "y"

	fmt.Println(s.slotsHeader)

	discovery, err := prompt(br, s.discoveryPrompt, "brainstorming")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := prompt(br, s.refinementPrompt, "openspec-explore")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := prompt(br, s.executionPrompt, "sdd-ask")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: execution: %w", err)
	}

	fmt.Println(s.chestHeader)

	chestPath, err := prompt(br, s.chestPrompt, "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: treasure_chest: %w", err)
	}

	return domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		UILanguage:         language,
		DocLanguage:        language,
		ChatLanguage:       language,
		CodeLanguage:       language,
		AdrEnabled:         adrEnabled,
		DiscoveryProvider:  discovery,
		RefinementProvider: refinement,
		ExecutionProvider:  execution,
		TreasureChestPath:  chestPath,
	}, nil
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

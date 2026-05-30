package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RunWizard prompts the user interactively for install configuration.
// Uses simple stdin prompts as a reliable fallback; replace with charmbracelet/huh
// for a richer TUI when desired.
func RunWizard() (domain.WizardConfig, error) {
	return runWizard(os.Stdin)
}

// runWizard is the testable core of RunWizard, reading from r instead of os.Stdin.
func runWizard(r io.Reader) (domain.WizardConfig, error) {
	br := bufio.NewReader(r)

	mode, err := prompt(br, "Mode (full/lightweight/minimal) [full]: ", "full")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}

	basePath, err := prompt(br, "Base path for analysis workspace [.analysis]: ", ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	language, err := promptValidated(br, "Artifact language (pt/en) [pt]: ", "pt", []string{"pt", "en"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: language: %w", err)
	}

	adrRaw, err := promptValidated(br, "Enable ADR generation at mission end? (yes/no) [yes]: ", "yes", []string{"yes", "no", "y", "n"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}
	adrEnabled := adrRaw == "yes" || adrRaw == "y"

	fmt.Println("\nSlot providers — which skill fills each mission role:")

	discovery, err := prompt(br, "  Ranger / discovery provider [brainstorming]: ", "brainstorming")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: discovery: %w", err)
	}

	refinement, err := prompt(br, "  Arquivista / refinement provider [openspec-explore]: ", "openspec-explore")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: refinement: %w", err)
	}

	execution, err := prompt(br, "  Sniper / execution provider [sdd-ask]: ", "sdd-ask")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: execution: %w", err)
	}

	fmt.Println("\nTreasure chest — optional offline knowledge source for all slots:")

	chestPath, err := prompt(br, "  Knowledge source path (e.g. .sdd/source) [leave empty to skip]: ", "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: treasure_chest: %w", err)
	}

	return domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		Language:           language,
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

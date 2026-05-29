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

	basePath, err := prompt(br, "Base path for .analysis/ workspace [.]: ", ".")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}

	provider, err := prompt(br, "Default provider (leave blank to skip): ", "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: provider: %w", err)
	}

	return domain.WizardConfig{
		Mode:     mode,
		BasePath: basePath,
		Provider: provider,
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

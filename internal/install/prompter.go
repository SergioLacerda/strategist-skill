package install

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Prompter abstracts interactive prompts so the wizard logic is independent of
// the underlying UI (TUI with arrow-keys vs plain text for CI/tests).
type Prompter interface {
	// Select shows a list of options and returns the chosen value.
	// defaultVal is pre-selected.
	Select(title, defaultVal string, options []string) (string, error)

	// Input shows a text field with defaultVal pre-filled.
	// Returns defaultVal when the user submits an empty line.
	Input(title, defaultVal string) (string, error)

	// SelectOrInput shows a Select with options plus customLabel as the last
	// entry. When the user picks customLabel, a free-text Input is shown.
	// defaultVal is pre-selected in the list.
	SelectOrInput(title, defaultVal string, options []string, customLabel string) (string, error)
}

// TextPrompter implements Prompter using plain stdin/stdout — used in tests
// and when stdout is not a TTY.
type TextPrompter struct {
	r *bufio.Reader
}

// NewTextPrompter wraps r in a TextPrompter.
func NewTextPrompter(r io.Reader) Prompter {
	return &TextPrompter{r: bufio.NewReader(r)}
}

// Select prints title and reads a line, returning the matching canonical option.
func (p *TextPrompter) Select(title, defaultVal string, options []string) (string, error) {
	for {
		fmt.Print(title)
		line, err := p.readTrimmedLine()
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return defaultVal, nil
		}
		if value, ok := matchOption(line, options); ok {
			return value, nil
		}
		fmt.Printf("  Invalid value %q. Accepted: %s\n", line, strings.Join(options, ", "))
	}
}

func (p *TextPrompter) readTrimmedLine() (string, error) {
	line, err := p.r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func matchOption(line string, options []string) (string, bool) {
	for _, o := range options {
		if strings.EqualFold(line, o) {
			return o, true
		}
	}
	return "", false
}

// Input prints title and reads a line, returning defaultVal on empty input.
func (p *TextPrompter) Input(title, defaultVal string) (string, error) {
	fmt.Print(title)
	line, err := p.readTrimmedLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// SelectOrInput in text mode simply falls through to Input — the caller types
// any value they want, including one of the listed options.
func (p *TextPrompter) SelectOrInput(title, defaultVal string, _ []string, _ string) (string, error) {
	return p.Input(title, defaultVal)
}

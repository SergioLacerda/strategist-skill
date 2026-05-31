package install

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// customSentinel is the internal value used to detect when the user picks
// "type custom..." in SelectOrInput.
const customSentinel = "\x00__custom__"

// TUIPrompter implements Prompter using charmbracelet/huh for interactive
// terminal UIs with arrow-key navigation.
type TUIPrompter struct {
	// runFn executes a single huh field. Defaults to huh.Run; overridable in tests.
	runFn func(huh.Field) error
}

// NewTUIPrompter returns a Prompter backed by huh.
func NewTUIPrompter() Prompter {
	return &TUIPrompter{runFn: huh.Run}
}

// Select renders an arrow-key selection list and returns the chosen option.
func (p *TUIPrompter) Select(title, defaultVal string, options []string) (string, error) {
	result := defaultVal
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	if err := p.runFn(
		huh.NewSelect[string]().
			Title(title).
			Options(opts...).
			Value(&result),
	); err != nil {
		return "", fmt.Errorf("select %q: %w", title, err)
	}
	return result, nil
}

// Input renders a free-text field with defaultVal pre-filled.
func (p *TUIPrompter) Input(title, defaultVal string) (string, error) {
	result := defaultVal
	if err := p.runFn(
		huh.NewInput().
			Title(title).
			Value(&result),
	); err != nil {
		return "", fmt.Errorf("input %q: %w", title, err)
	}
	return result, nil
}

// SelectOrInput renders a selection list that includes customLabel as the last entry.
// When the user picks customLabel, a free-text Input is shown for arbitrary input.
func (p *TUIPrompter) SelectOrInput(title, defaultVal string, options []string, customLabel string) (string, error) {
	opts := make([]huh.Option[string], len(options)+1)
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	opts[len(options)] = huh.NewOption(customLabel, customSentinel)

	result := defaultVal
	if err := p.runFn(
		huh.NewSelect[string]().
			Title(title).
			Options(opts...).
			Value(&result),
	); err != nil {
		return "", fmt.Errorf("select %q: %w", title, err)
	}

	if result != customSentinel {
		return result, nil
	}

	// User chose "type custom..." — show a free-text input.
	var custom string
	if err := p.runFn(
		huh.NewInput().
			Title(title).
			Value(&custom),
	); err != nil {
		return "", fmt.Errorf("input %q: %w", title, err)
	}
	return custom, nil
}

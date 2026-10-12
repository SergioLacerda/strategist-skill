package wizard

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// CustomSentinel is the internal value used to detect when the user picks
// "type custom..." in SelectOrInput.
const CustomSentinel = "\x00__custom__"

// TUIPrompter implements Prompter using charmbracelet/huh for interactive
// terminal UIs with arrow-key navigation.
type TUIPrompter struct {
	// RunFn executes a single huh field. Defaults to huh.Run; overridable in tests.
	RunFn func(huh.Field) error
}

// NewTUIPrompter returns a Prompter backed by huh.
func NewTUIPrompter() Prompter {
	return &TUIPrompter{RunFn: huh.Run}
}

// Select renders an arrow-key selection list and returns the chosen option.
func (p *TUIPrompter) Select(title, defaultVal string, options []string) (string, error) {
	result := defaultVal
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	if err := p.RunFn(
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
	if err := p.RunFn(
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
	labeled := make([]PromptOption, len(options))
	for i, o := range options {
		labeled[i] = PromptOption{Label: o, Value: o}
	}
	return p.selectOrInputWithLabels(title, defaultVal, labeled, customLabel)
}

func (p *TUIPrompter) selectOrInputWithLabels(title, defaultVal string, options []PromptOption, customLabel string) (string, error) {
	opts := make([]huh.Option[string], len(options)+1)
	for i, option := range options {
		opts[i] = huh.NewOption(option.Label, option.Value)
	}
	opts[len(options)] = huh.NewOption(customLabel, CustomSentinel)

	result := defaultVal
	if err := p.RunFn(
		huh.NewSelect[string]().
			Title(title).
			Options(opts...).
			Value(&result),
	); err != nil {
		return "", fmt.Errorf("select %q: %w", title, err)
	}

	if result != CustomSentinel {
		return result, nil
	}

	// User chose "type custom..." — show a free-text input.
	var custom string
	if err := p.RunFn(
		huh.NewInput().
			Title(title).
			Value(&custom),
	); err != nil {
		return "", fmt.Errorf("input %q: %w", title, err)
	}
	return custom, nil
}

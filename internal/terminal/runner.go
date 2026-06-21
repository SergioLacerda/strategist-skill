// Package terminal provides animated progress indicators for the Strategist CLI.
package terminal

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// SendFn reports progress to the bar: done steps out of total.
type SendFn func(done, total int)

// RunSpellCharge displays a Spell Charge progress bar while fn executes.
// fn receives a SendFn to report step progress.
// In non-TTY environments (CI/CD, pipes) fn runs directly with no animation.
func RunSpellCharge(ctx context.Context, label string, totalSteps int, fn func(SendFn) error) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fn(func(_, _ int) {})
	}

	m := newSpellCharge(label, totalSteps)
	prog := tea.NewProgram(m, tea.WithContext(ctx))

	errCh := make(chan error, 1)
	go func() {
		err := fn(func(done, total int) {
			prog.Send(ProgressMsg{Done: done, Total: total})
		})
		prog.Send(DoneMsg{Err: err})
		errCh <- err
	}()

	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("spell charge program: %w", err)
	}
	return <-errCh
}

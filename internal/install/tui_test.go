package install

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nopRun simulates a successful huh.Run that leaves the field value unchanged
// (the Prompter methods pre-load the result with defaultVal via Value(&result)).
func nopRun(_ huh.Field) error { return nil }

func errRun(_ huh.Field) error { return errors.New("terminal unavailable") }

func tuiP() *TUIPrompter { return &TUIPrompter{runFn: nopRun} }

// --- Select ---

func TestTUIPrompter_Select_ReturnsDefault(t *testing.T) {
	t.Parallel()
	val, err := tuiP().Select("language", "en", []string{"en", "pt-BR"})
	require.NoError(t, err)
	assert.Equal(t, "en", val)
}

func TestTUIPrompter_Select_Error(t *testing.T) {
	t.Parallel()
	p := &TUIPrompter{runFn: errRun}
	_, err := p.Select("language", "en", []string{"en", "pt-BR"})
	require.Error(t, err)
	require.ErrorContains(t, err, "select")
	require.ErrorContains(t, err, "language")
}

// --- Input ---

func TestTUIPrompter_Input_ReturnsDefault(t *testing.T) {
	t.Parallel()
	val, err := tuiP().Input("base path", ".analysis")
	require.NoError(t, err)
	assert.Equal(t, ".analysis", val)
}

func TestTUIPrompter_Input_Error(t *testing.T) {
	t.Parallel()
	p := &TUIPrompter{runFn: errRun}
	_, err := p.Input("base path", ".analysis")
	require.Error(t, err)
	require.ErrorContains(t, err, "input")
	require.ErrorContains(t, err, "base path")
}

// --- SelectOrInput ---

func TestTUIPrompter_SelectOrInput_ReturnsDefault(t *testing.T) {
	t.Parallel()
	// nopRun leaves result as defaultVal ("brainstorming" ≠ customSentinel) → returns it directly.
	val, err := tuiP().SelectOrInput("provider", "brainstorming", []string{"brainstorming"}, "(digitar outro...)")
	require.NoError(t, err)
	assert.Equal(t, "brainstorming", val)
}

func TestTUIPrompter_SelectOrInput_CustomPath(t *testing.T) {
	t.Parallel()
	// Setting defaultVal = customSentinel makes nopRun leave result as customSentinel,
	// triggering the second Input prompt, which also nop-runs and returns "".
	val, err := tuiP().SelectOrInput("provider", customSentinel, []string{"brainstorming"}, "(digitar outro...)")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestTUIPrompter_SelectOrInput_SelectError(t *testing.T) {
	t.Parallel()
	p := &TUIPrompter{runFn: errRun}
	_, err := p.SelectOrInput("provider", "brainstorming", []string{"brainstorming"}, "(digitar outro...)")
	require.Error(t, err)
	require.ErrorContains(t, err, "select")
}

func TestTUIPrompter_SelectOrInput_InputError(t *testing.T) {
	t.Parallel()
	// First call (Select) succeeds, second call (Input) fails.
	calls := 0
	p := &TUIPrompter{runFn: func(_ huh.Field) error {
		calls++
		if calls == 1 {
			return nil // Select succeeds; result stays as defaultVal = customSentinel
		}
		return errors.New("terminal unavailable")
	}}
	_, err := p.SelectOrInput("provider", customSentinel, []string{"brainstorming"}, "(digitar outro...)")
	require.Error(t, err)
	require.ErrorContains(t, err, "input")
	assert.Equal(t, 2, calls)
}

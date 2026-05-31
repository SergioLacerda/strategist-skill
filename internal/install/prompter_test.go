package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textPrompter(input string) Prompter {
	return NewTextPrompter(strings.NewReader(input))
}

func TestTextPrompter_Select_Default(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("\n").Select("lang", "en", []string{"en", "pt-BR"})
	require.NoError(t, err)
	assert.Equal(t, "en", val)
}

func TestTextPrompter_Select_ValidValue(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("pt-BR\n").Select("lang", "en", []string{"en", "pt-BR"})
	require.NoError(t, err)
	assert.Equal(t, "pt-BR", val)
}

func TestTextPrompter_Select_CaseInsensitive(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("PT-BR\n").Select("lang", "en", []string{"en", "pt-BR"})
	require.NoError(t, err)
	assert.Equal(t, "pt-BR", val)
}

func TestTextPrompter_Select_RejectsInvalidThenAccepts(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("invalid\nbad\nen\n").Select("lang", "pt-BR", []string{"pt-BR", "en"})
	require.NoError(t, err)
	assert.Equal(t, "en", val)
}

func TestTextPrompter_Input_Default(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("\n").Input("base path", ".analysis")
	require.NoError(t, err)
	assert.Equal(t, ".analysis", val)
}

func TestTextPrompter_Input_CustomValue(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("/workspace\n").Input("base path", ".analysis")
	require.NoError(t, err)
	assert.Equal(t, "/workspace", val)
}

func TestTextPrompter_SelectOrInput_Default(t *testing.T) {
	t.Parallel()
	// SelectOrInput in text mode accepts any value — defaults when empty.
	val, err := textPrompter("\n").SelectOrInput("provider", "brainstorming", []string{"brainstorming"}, "(digitar outro...)")
	require.NoError(t, err)
	assert.Equal(t, "brainstorming", val)
}

func TestTextPrompter_SelectOrInput_CustomValue(t *testing.T) {
	t.Parallel()
	val, err := textPrompter("my-skill\n").SelectOrInput("provider", "brainstorming", []string{"brainstorming"}, "(digitar outro...)")
	require.NoError(t, err)
	assert.Equal(t, "my-skill", val)
}

func TestTextPrompter_Select_EOF(t *testing.T) {
	t.Parallel()
	_, err := textPrompter("").Select("lang", "en", []string{"en", "pt-BR"})
	require.Error(t, err)
}

func TestTextPrompter_Input_EOF(t *testing.T) {
	t.Parallel()
	_, err := textPrompter("").Input("path", ".analysis")
	require.Error(t, err)
}

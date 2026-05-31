package install

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBufReader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func TestRunWizard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		wantMode       string
		wantBase       string
		wantLanguage   string
		wantAdrEnabled bool
		wantDiscovery  string
		wantRefinement string
		wantExecution  string
		wantChestPath  string
	}{
		{
			name: "all defaults (empty lines)",
			// 9 prompts: lang/mode/base/language/adr/discovery/refinement/execution/chest
			input:          "\n\n\n\n\n\n\n\n\n",
			wantMode:       "full",
			wantBase:       ".analysis",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
			wantChestPath:  "",
		},
		{
			name:           "english ui language, custom slots with chest",
			input:          "en\nlightweight\n/workspace\npt\nyes\nbrainstorming\narchivist\nsdd-ask-full\n.sdd/source\n",
			wantMode:       "lightweight",
			wantBase:       "/workspace",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "archivist",
			wantExecution:  "sdd-ask-full",
			wantChestPath:  ".sdd/source",
		},
		{
			name:           "pt-br ui language",
			input:          "pt-br\nminimal\n.\nen\nno\n\n\n\n\n",
			wantMode:       "minimal",
			wantBase:       ".",
			wantLanguage:   "en",
			wantAdrEnabled: false,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
			wantChestPath:  "",
		},
		{
			name:           "short form y accepted for adr",
			input:          "\nfull\n.\npt\ny\n\n\n\n\n",
			wantMode:       "full",
			wantBase:       ".",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
			wantChestPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantLanguage, wc.UILanguage)
			assert.Equal(t, tt.wantAdrEnabled, wc.AdrEnabled)
			assert.Equal(t, tt.wantDiscovery, wc.DiscoveryProvider)
			assert.Equal(t, tt.wantRefinement, wc.RefinementProvider)
			assert.Equal(t, tt.wantExecution, wc.ExecutionProvider)
			assert.Equal(t, tt.wantChestPath, wc.TreasureChestPath)
		})
	}
}

func TestPromptValidated_RejectsInvalidThenAccepts(t *testing.T) {
	t.Parallel()
	// Primeiros dois inputs são inválidos; o terceiro é válido.
	input := "invalid\nbad\nen\n"
	br := newBufReader(input)
	val, err := promptValidated(br, "lang: ", "pt", []string{"pt", "en"})
	require.NoError(t, err)
	assert.Equal(t, "en", val)
}

func TestPromptValidated_DefaultOnEmpty(t *testing.T) {
	t.Parallel()
	br := newBufReader("\n")
	val, err := promptValidated(br, "lang: ", "pt", []string{"pt", "en"})
	require.NoError(t, err)
	assert.Equal(t, "pt", val)
}

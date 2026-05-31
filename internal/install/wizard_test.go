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
		name             string
		input            string
		wantUILanguage   string
		wantDocLanguage  string
		wantChatLanguage string
		wantCodeLanguage string
		wantMode         string
		wantBase         string
		wantAdrEnabled   bool
		wantDiscovery    string
		wantRefinement   string
		wantExecution    string
		wantChestPath    string
	}{
		{
			name: "all defaults (empty lines)",
			// 11 prompts: uiLang/docLang/chatLang/codeLang/mode/base/adr/discovery/refinement/execution/chest
			input:            "\n\n\n\n\n\n\n\n\n\n\n",
			wantUILanguage:   "en",
			wantDocLanguage:  "en",
			wantChatLanguage: "en",
			wantCodeLanguage: "en",
			wantMode:         "full",
			wantBase:         ".analysis",
			wantAdrEnabled:   true,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "openspec-explore",
			wantExecution:    "sdd-ask",
			wantChestPath:    "",
		},
		{
			name:             "en ui, custom languages and slots with chest",
			input:            "en\nen\npt-BR\nen\nlightweight\n/workspace\nyes\nbrainstorming\narchivist\nsdd-ask-full\n.sdd/source\n",
			wantUILanguage:   "en",
			wantDocLanguage:  "en",
			wantChatLanguage: "pt-BR",
			wantCodeLanguage: "en",
			wantMode:         "lightweight",
			wantBase:         "/workspace",
			wantAdrEnabled:   true,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "archivist",
			wantExecution:    "sdd-ask-full",
			wantChestPath:    ".sdd/source",
		},
		{
			name:             "pt-BR ui language, ADR disabled",
			input:            "pt-BR\nen\npt-BR\nen\nminimal\n.\nno\n\n\n\n\n",
			wantUILanguage:   "pt-BR",
			wantDocLanguage:  "en",
			wantChatLanguage: "pt-BR",
			wantCodeLanguage: "en",
			wantMode:         "minimal",
			wantBase:         ".",
			wantAdrEnabled:   false,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "openspec-explore",
			wantExecution:    "sdd-ask",
			wantChestPath:    "",
		},
		{
			name:             "short form y accepted for adr",
			input:            "\n\n\n\nfull\n.\ny\n\n\n\n\n",
			wantUILanguage:   "en",
			wantDocLanguage:  "en",
			wantChatLanguage: "en",
			wantCodeLanguage: "en",
			wantMode:         "full",
			wantBase:         ".",
			wantAdrEnabled:   true,
			wantDiscovery:    "brainstorming",
			wantRefinement:   "openspec-explore",
			wantExecution:    "sdd-ask",
			wantChestPath:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantUILanguage, wc.UILanguage)
			assert.Equal(t, tt.wantDocLanguage, wc.DocLanguage)
			assert.Equal(t, tt.wantChatLanguage, wc.ChatLanguage)
			assert.Equal(t, tt.wantCodeLanguage, wc.CodeLanguage)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
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

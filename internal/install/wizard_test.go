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
	}{
		{
			name:           "all defaults (empty lines)",
			input:          "\n\n\n\n\n\n\n",
			wantMode:       "full",
			wantBase:       ".analysis",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
		},
		{
			name:           "custom slots",
			input:          "lightweight\n/workspace\npt\nyes\nbrainstorming\narchivist\nsdd-ask-full\n",
			wantMode:       "lightweight",
			wantBase:       "/workspace",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "archivist",
			wantExecution:  "sdd-ask-full",
		},
		{
			name:           "english language ADR disabled",
			input:          "minimal\n.\nen\nno\n\n\n\n",
			wantMode:       "minimal",
			wantBase:       ".",
			wantLanguage:   "en",
			wantAdrEnabled: false,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
		},
		{
			name:           "short form y accepted for adr",
			input:          "full\n.\npt\ny\n\n\n\n",
			wantMode:       "full",
			wantBase:       ".",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
			wantDiscovery:  "brainstorming",
			wantRefinement: "openspec-explore",
			wantExecution:  "sdd-ask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantLanguage, wc.Language)
			assert.Equal(t, tt.wantAdrEnabled, wc.AdrEnabled)
			assert.Equal(t, tt.wantDiscovery, wc.DiscoveryProvider)
			assert.Equal(t, tt.wantRefinement, wc.RefinementProvider)
			assert.Equal(t, tt.wantExecution, wc.ExecutionProvider)
		})
	}
}

func TestPromptValidated_RejectsInvalidThenAccepts(t *testing.T) {
	t.Parallel()
	// First two inputs are invalid; third is valid.
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

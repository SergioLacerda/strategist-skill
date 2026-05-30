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
		wantProv       string
		wantLanguage   string
		wantAdrEnabled bool
	}{
		{
			name:           "all defaults (empty lines)",
			input:          "\n\n\n\n\n",
			wantMode:       "full",
			wantBase:       ".analysis",
			wantProv:       "",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
		},
		{
			name:           "custom values PT ADR enabled",
			input:          "lightweight\n/workspace\nclaude\npt\nyes\n",
			wantMode:       "lightweight",
			wantBase:       "/workspace",
			wantProv:       "claude",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
		},
		{
			name:           "english language ADR disabled",
			input:          "minimal\n.\n\nen\nno\n",
			wantMode:       "minimal",
			wantBase:       ".",
			wantProv:       "",
			wantLanguage:   "en",
			wantAdrEnabled: false,
		},
		{
			name:           "short form y/n accepted",
			input:          "full\n.\n\npt\ny\n",
			wantMode:       "full",
			wantBase:       ".",
			wantProv:       "",
			wantLanguage:   "pt",
			wantAdrEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wc, err := runWizard(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, wc.Mode)
			assert.Equal(t, tt.wantBase, wc.BasePath)
			assert.Equal(t, tt.wantProv, wc.Provider)
			assert.Equal(t, tt.wantLanguage, wc.Language)
			assert.Equal(t, tt.wantAdrEnabled, wc.AdrEnabled)
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

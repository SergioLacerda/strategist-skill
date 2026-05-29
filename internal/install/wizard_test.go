package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWizard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		wantMode string
		wantBase string
		wantProv string
	}{
		{
			name:     "all defaults (empty lines)",
			input:    "\n\n\n",
			wantMode: "full",
			wantBase: ".",
			wantProv: "",
		},
		{
			name:     "custom values",
			input:    "lightweight\n/workspace\nclaude\n",
			wantMode: "lightweight",
			wantBase: "/workspace",
			wantProv: "claude",
		},
		{
			name:     "mode override, base default, no provider",
			input:    "minimal\n\n\n",
			wantMode: "minimal",
			wantBase: ".",
			wantProv: "",
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
		})
	}
}

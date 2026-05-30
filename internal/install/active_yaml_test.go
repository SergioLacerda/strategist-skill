package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteActiveYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         domain.WizardConfig
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "full mode with custom slots",
			cfg: domain.WizardConfig{
				Mode:               "full",
				BasePath:           ".analysis",
				Language:           "pt",
				AdrEnabled:         true,
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "openspec-explore",
				ExecutionProvider:  "sdd-ask",
			},
			wantContain: []string{
				"mode: full",
				"base_path: .analysis",
				"language: pt",
				"adr_enabled: true",
				"discovery: brainstorming",
				"refinement: openspec-explore",
				"execution: sdd-ask",
			},
		},
		{
			name: "minimal mode ADR disabled english",
			cfg: domain.WizardConfig{
				Mode:               "minimal",
				BasePath:           ".",
				Language:           "en",
				AdrEnabled:         false,
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "archivist",
				ExecutionProvider:  "sdd-ask-full",
			},
			wantContain: []string{
				"mode: minimal",
				"language: en",
				"adr_enabled: false",
				"refinement: archivist",
				"execution: sdd-ask-full",
			},
			wantAbsent: []string{"roles_config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, writeActiveYAML(dir, tt.cfg))
			data, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
			require.NoError(t, err)
			s := string(data)
			for _, want := range tt.wantContain {
				assert.Contains(t, s, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, s, absent)
			}
		})
	}
}

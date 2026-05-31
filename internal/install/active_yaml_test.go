package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want string
	}{
		{".sdd/source", "source"},
		{"source", "source"},
		{"/absolute/path/to/chest", "chest"},
		{"trailing/slash/", "slash"},
		{"nodslash", "nodslash"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, treasureChestID(tt.path))
		})
	}
}

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
				UILanguage:         "en",
				DocLanguage:        "en",
				ChatLanguage:       "pt-BR",
				CodeLanguage:       "en",
				AdrEnabled:         true,
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "openspec-explore",
				ExecutionProvider:  "sdd-ask",
			},
			wantContain: []string{
				"mode: full",
				"base_path: .analysis",
				"language:",
				"  ui: en",
				"  docs: en",
				"  chat: pt-BR",
				"  code: en",
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
				UILanguage:         "en",
				DocLanguage:        "en",
				ChatLanguage:       "en",
				CodeLanguage:       "en",
				AdrEnabled:         false,
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "archivist",
				ExecutionProvider:  "sdd-ask-full",
			},
			wantContain: []string{
				"mode: minimal",
				"language:",
				"  ui: en",
				"adr_enabled: false",
				"refinement: archivist",
				"execution: sdd-ask-full",
			},
			wantAbsent: []string{"roles_config", "language: en"},
		},
		{
			name: "with treasure chest path",
			cfg: domain.WizardConfig{
				Mode:               "full",
				BasePath:           ".analysis",
				UILanguage:         "en",
				DocLanguage:        "en",
				ChatLanguage:       "pt-BR",
				CodeLanguage:       "en",
				AdrEnabled:         true,
				DiscoveryProvider:  "brainstorming",
				RefinementProvider: "openspec-explore",
				ExecutionProvider:  "sdd-ask",
				TreasureChestPath:  ".sdd/source",
			},
			wantContain: []string{
				"treasure_chests:",
				"id: source",
				"path: .sdd/source",
				"scope: all",
			},
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

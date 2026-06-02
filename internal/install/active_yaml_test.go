package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteKnowledgeIndexSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		initialKI   string
		cfg         domain.WizardConfig
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "no chest path — file unchanged",
			initialKI:   "sources: []\n",
			cfg:         domain.WizardConfig{TreasureChestPath: ""},
			wantContain: []string{"sources: []"},
		},
		{
			name:      "chest path replaces placeholder",
			initialKI: "sources: []\n",
			cfg:       domain.WizardConfig{TreasureChestPath: ".sdd/source"},
			wantContain: []string{
				"id: source",
				"path: .sdd/source",
				"tags: [all]",
			},
			wantAbsent: []string{"sources: []"},
		},
		{
			name:      "preserves surrounding comments",
			initialKI: "# preamble\nsources: []\n# postamble\n",
			cfg:       domain.WizardConfig{TreasureChestPath: "docs/specs"},
			wantContain: []string{
				"# preamble",
				"id: specs",
				"# postamble",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			kiPath := filepath.Join(dir, "knowledge.index.yaml")
			require.NoError(t, os.WriteFile(kiPath, []byte(tt.initialKI), 0o644))
			require.NoError(t, writeKnowledgeIndexSource(dir, tt.cfg))
			data, err := os.ReadFile(kiPath)
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

func TestWriteKnowledgeIndexSource_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "read knowledge.index.yaml")
}

func TestWriteKnowledgeIndexSource_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o444)) // read-only
	t.Cleanup(func() { _ = os.Chmod(kiPath, 0o644) })
	err := writeKnowledgeIndexSource(dir, domain.WizardConfig{TreasureChestPath: ".sdd/source"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "write knowledge.index.yaml")
}

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
				Mode:               "pragmatic",
				BasePath:           ".analysis",
				MissionMode:        "analise",
				DoneScope:          "analise",
				ApplyChanges:       true, // should be normalized to false for analise
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
				"mode: pragmatic",
				"base_path: .analysis",
				"roles_config: roles/default.yaml",
				"language:",
				"  ui: en",
				"  docs: en",
				"  chat: pt-BR",
				"  code: en",
				"adr_enabled: true",
				"mission_mode: analise",
				"escopo_done: analise",
				"aplicar_alteracoes: false",
				"discovery: brainstorming",
				"refinement: openspec-explore",
				"execution: sdd-ask",
			},
		},
		{
			name: "minimal mode ADR disabled english",
			cfg: domain.WizardConfig{
				Mode:               "epic",
				BasePath:           ".",
				MissionMode:        "entrega_executada",
				DoneScope:          "entrega",
				ApplyChanges:       true,
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
				"mode: epic",
				"roles_config: roles/default.yaml",
				"language:",
				"  ui: en",
				"adr_enabled: false",
				"mission_mode: entrega_executada",
				"escopo_done: entrega",
				"aplicar_alteracoes: true",
				"refinement: archivist",
				"execution: sdd-ask-full",
			},
			wantAbsent: []string{"language: en"},
		},
		{
			name: "with treasure chest path",
			cfg: domain.WizardConfig{
				Mode:               "pragmatic",
				BasePath:           ".analysis",
				MissionMode:        "analise",
				DoneScope:          "analise",
				ApplyChanges:       false,
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

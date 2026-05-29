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
	tests := []struct {
		name     string
		cfg      domain.WizardConfig
		wantKeys []string
	}{
		{
			name:     "full mode with provider",
			cfg:      domain.WizardConfig{Mode: "full", BasePath: ".analysis", Provider: "claude"},
			wantKeys: []string{"mode: full", "base_path: .analysis", "provider: claude"},
		},
		{
			name:     "minimal mode without provider",
			cfg:      domain.WizardConfig{Mode: "minimal", BasePath: ".", Provider: ""},
			wantKeys: []string{"mode: minimal", "base_path: ."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, writeActiveYAML(dir, tt.cfg))
			data, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
			require.NoError(t, err)
			for _, key := range tt.wantKeys {
				assert.Contains(t, string(data), key)
			}
			if tt.cfg.Provider == "" {
				assert.NotContains(t, string(data), "provider:")
			}
		})
	}
}

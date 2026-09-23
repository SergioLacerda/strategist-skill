package policy_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteScopeFromActive(t *testing.T) {
	t.Parallel()
	project := filepath.Join(t.TempDir(), "project")
	runtime := filepath.Join(project, ".strategist")

	tests := []struct {
		name    string
		cfg     domain.ActiveConfig
		want    policy.WriteScope
		wantErr string
	}{
		{
			name:    "empty base_path",
			cfg:     domain.ActiveConfig{},
			wantErr: "active.yaml: base_path is empty",
		},
		{
			name: "defaults documentation roots to docs",
			cfg:  domain.ActiveConfig{BasePath: ".analysis"},
			want: policy.WriteScope{
				AnalysisRoot:       filepath.Join(project, ".analysis"),
				DocumentationRoots: []string{filepath.Join(project, "docs")},
				RuntimeRoot:        runtime,
			},
		},
		{
			name: "declared documentation roots",
			cfg:  domain.ActiveConfig{BasePath: ".analysis", DocumentationRoots: []string{"handbook", "site/docs"}},
			want: policy.WriteScope{
				AnalysisRoot:       filepath.Join(project, ".analysis"),
				DocumentationRoots: []string{filepath.Join(project, "handbook"), filepath.Join(project, "site/docs")},
				RuntimeRoot:        runtime,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := policy.WriteScopeFromActive(runtime, tc.cfg)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileDomain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		setup   func(t testing.TB, dir string)
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name: "empty load_always and load_by_task_type",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-domain/1.0", a["schema"])
				assertSourceStats(t, a)
				assert.NotNil(t, a["load_always"])
				assert.NotNil(t, a["load_by_task_type"])
			},
		},
		{
			name: "load_always with existing file",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "roles.yaml"), []byte("roles: true\n"), 0o644))
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always:\n  - roles.yaml\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				la, ok := a["load_always"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, la, "roles.yaml")
			},
		},
		{
			name: "missing file in load_always is skipped",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always:\n  - missing.yaml\nload_by_task_type: {}\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				la, ok := a["load_always"].(map[string]any)
				require.True(t, ok)
				assert.NotContains(t, la, "missing.yaml")
			},
		},
		{
			name: "load_by_task_type with task types",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "arch.yaml"), []byte("arch: true\n"), 0o644))
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type:\n  analysis:\n    - arch.yaml\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				lbtt := a["load_by_task_type"].(map[string]any)
				assert.Contains(t, lbtt, "analysis")
			},
		},
		{
			name: "missing index.yaml returns error",
			setup: func(t testing.TB, _ string) {
				t.Helper()
			},
			wantErr: true,
		},
		{
			name: "invalid YAML in index.yaml returns error",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: [invalid yaml: : :\n"),
					0o644,
				))
			},
			wantErr: true,
		},
		{
			name: "load_by_task_type with missing file is skipped",
			setup: func(t testing.TB, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "index.yaml"),
					[]byte("load_always: []\nload_by_task_type:\n  analysis:\n    - nonexistent.yaml\n"),
					0o644,
				))
			},
			check: func(t *testing.T, a map[string]any) {
				lbtt := a["load_by_task_type"].(map[string]any)
				analysis := lbtt["analysis"].(map[string]any)
				assert.Empty(t, analysis)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tt.setup(t, dir)
			out := filepath.Join(dir, ".compiled", ".domain.gz")
			err := compile.Domain(dir, out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out)
			var artifact map[string]any
			testutil.ReadGzJSON(t, out, &artifact)
			if tt.check != nil {
				tt.check(t, artifact)
			}
		})
	}
}

func TestCompileDomain_InvalidYAMLInLoadAlways(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(": invalid\n  yaml:"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "index.yaml"),
		[]byte("load_always:\n  - bad.yaml\nload_by_task_type: {}\n"),
		0o644,
	))
	err := compile.Domain(dir, filepath.Join(dir, ".compiled", ".domain.gz"))
	require.Error(t, err)
}

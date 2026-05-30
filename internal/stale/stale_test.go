package stale_test

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStale(t *testing.T) {
	t.Parallel()
	checker := stale.Checker{}

	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string
		wantStale bool
		wantErr   bool
	}{
		{
			name: "absent artifact is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "absent.gz")
			},
			wantStale: true,
		},
		{
			name: "missing manifest is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				return art
			},
			wantStale: true,
		},
		{
			name: "fresh artifact with empty sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
		},
		{
			name: "fresh artifact with nil sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				// No "sources" key at all — should be treated as empty
				testutil.WriteGzJSON(t, art, map[string]any{})
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
		},
		{
			name: "source newer than recorded mtime is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "active.yaml")
				require.NoError(t, os.WriteFile(src, []byte("mode: full"), 0o644))
				past := time.Now().Add(-1 * time.Hour)
				require.NoError(t, os.Chtimes(src, past, past))
				now := time.Now()
				require.NoError(t, os.Chtimes(src, now, now))
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{
					"sources": map[string]int64{src: past.Unix()},
				})
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: true,
		},
		{
			name: "source gone after compile is stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				src := filepath.Join(dir, "gone.yaml")
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{
					"sources": map[string]int64{src: time.Now().Unix()},
				})
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: true,
		},
		{
			name: "corrupt gzip artifact returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				require.NoError(t, os.WriteFile(art, []byte("not gzip data"), 0o644))
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
		{
			name: "valid gzip with invalid JSON returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				// Write valid gzip wrapping invalid JSON bytes
				f, err := os.Create(art)
				require.NoError(t, err)
				gz := gzip.NewWriter(f)
				_, err = gz.Write([]byte("{not valid json"))
				require.NoError(t, err)
				require.NoError(t, gz.Close())
				require.NoError(t, f.Close())
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{})
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
		{
			name: "artifact in unreadable directory returns error",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				if os.Getuid() == 0 {
					t.Skip("permission tests do not apply when running as root")
				}
				subdir := filepath.Join(dir, "locked")
				require.NoError(t, os.Mkdir(subdir, 0o755))
				art := filepath.Join(subdir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				testutil.WriteGzJSON(t, filepath.Join(subdir, ".manifest.gz"), map[string]any{})
				// Remove execute permission so stat of files inside fails with EACCES
				require.NoError(t, os.Chmod(subdir, 0o000))
				t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })
				return art
			},
			wantStale: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			artifactPath := tt.setup(t, dir)
			got, err := checker.IsStale(artifactPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStale, got)
		})
	}
}

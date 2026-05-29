package tests_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGzJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestIsStale(t *testing.T) {
	t.Parallel()
	checker := stale.Checker{}

	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string // returns artifact path
		wantStale bool
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
				writeGzJSON(t, art, map[string]interface{}{"sources": map[string]int64{}})
				// no .manifest.gz written
				return art
			},
			wantStale: true,
		},
		{
			name: "fresh artifact with empty sources is not stale",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{"sources": map[string]int64{}})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
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

				// Record an mtime in the past
				past := time.Now().Add(-1 * time.Hour)
				require.NoError(t, os.Chtimes(src, past, past))

				// Then touch the file to update its mtime
				now := time.Now()
				require.NoError(t, os.Chtimes(src, now, now))

				art := filepath.Join(dir, ".config.gz")
				writeGzJSON(t, art, map[string]interface{}{
					"sources": map[string]int64{src: past.Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
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
				writeGzJSON(t, art, map[string]interface{}{
					"sources": map[string]int64{src: time.Now().Unix()},
				})
				writeGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]interface{}{})
				// src never created
				return art
			},
			wantStale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			artifactPath := tt.setup(t, dir)
			got, err := checker.IsStale(artifactPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStale, got)
		})
	}
}

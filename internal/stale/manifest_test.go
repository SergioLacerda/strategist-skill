package stale_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckReasons(t *testing.T) {
	t.Parallel()
	checker := stale.Checker{}

	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string) string
		wantStale  bool
		wantReason stale.Reason
	}{
		{
			name: "absent artifact",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				return filepath.Join(dir, "absent.gz")
			},
			wantStale:  true,
			wantReason: stale.ReasonMissingArtifact,
		},
		{
			name: "missing manifest",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				return art
			},
			wantStale:  true,
			wantReason: stale.ReasonMissingManifest,
		},
		{
			name: "manifest entry missing",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				testutil.WriteGzJSON(t, filepath.Join(dir, ".manifest.gz"), map[string]any{
					"artifacts": map[string]string{".domain.gz": "sha256:abc"},
				})
				return art
			},
			wantStale:  true,
			wantReason: stale.ReasonManifestEntryMissing,
		},
		{
			name: "artifact hash mismatch",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				writeManifestForArtifact(t, art)
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}, "extra": "changed"})
				return art
			},
			wantStale:  true,
			wantReason: stale.ReasonArtifactHashMismatch,
		},
		{
			name: "fresh with manifest hash",
			setup: func(t *testing.T, dir string) string {
				t.Helper()
				art := filepath.Join(dir, ".config.gz")
				testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
				writeManifestForArtifact(t, art)
				return art
			},
			wantStale:  false,
			wantReason: stale.ReasonFresh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			got, err := checker.Check(tt.setup(t, dir))
			require.NoError(t, err)
			assert.Equal(t, tt.wantStale, got.Stale)
			assert.Equal(t, tt.wantReason, got.Reason)
		})
	}
}

func TestCheckManifestArtifactHashUnavailable(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	checker := stale.Checker{}
	dir := t.TempDir()
	art := filepath.Join(dir, ".config.gz")
	testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
	writeManifestForArtifact(t, art)
	require.NoError(t, os.Chmod(art, 0o000))
	t.Cleanup(func() { _ = os.Chmod(art, 0o644) })

	got, err := checker.Check(art)
	require.NoError(t, err)
	assert.True(t, got.Stale)
	assert.Equal(t, stale.ReasonArtifactHashMismatch, got.Reason)
	assert.Contains(t, got.Detail, "unavailable")
}

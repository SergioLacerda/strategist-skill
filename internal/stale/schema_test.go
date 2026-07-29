package stale_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSourceSchemaHandling(t *testing.T) {
	t.Parallel()
	checker := stale.Checker{}

	t.Run("no sources metadata is fresh with detail", func(t *testing.T) {
		t.Parallel()
		art := filepath.Join(t.TempDir(), ".config.gz")
		testutil.WriteGzJSON(t, art, map[string]any{})
		writeManifestForArtifact(t, art)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.False(t, got.Stale)
		assert.Equal(t, stale.ReasonFresh, got.Reason)
		assert.Contains(t, got.Detail, "missing source metadata")
	})

	t.Run("empty sources is legacy fresh", func(t *testing.T) {
		t.Parallel()
		art := filepath.Join(t.TempDir(), ".config.gz")
		testutil.WriteGzJSON(t, art, map[string]any{"sources": map[string]int64{}})
		writeManifestForArtifact(t, art)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.False(t, got.Stale)
		assert.Contains(t, got.Detail, "legacy")
	})

	t.Run("invalid sources type returns error", func(t *testing.T) {
		t.Parallel()
		art := filepath.Join(t.TempDir(), ".config.gz")
		testutil.WriteGzJSON(t, art, map[string]any{"sources": "bad"})
		writeManifestForArtifact(t, art)
		_, err := checker.Check(art)
		require.Error(t, err)
		assert.ErrorContains(t, err, "sources")
	})

	t.Run("invalid source stats type returns error", func(t *testing.T) {
		t.Parallel()
		art := filepath.Join(t.TempDir(), ".config.gz")
		testutil.WriteGzJSON(t, art, map[string]any{"source_stats": "bad"})
		writeManifestForArtifact(t, art)
		_, err := checker.Check(art)
		require.Error(t, err)
		assert.ErrorContains(t, err, "source_stats")
	})
}

package stale_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckStrongSourceMetadata(t *testing.T) {
	t.Parallel()
	checker := stale.Checker{}

	t.Run("same second size mismatch is stale", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		require.NoError(t, os.WriteFile(src, []byte("mode: old"), 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{MTime: info.ModTime().Unix(), MTimeNS: info.ModTime().UnixNano(), Size: info.Size()}
		require.NoError(t, os.WriteFile(src, []byte("mode: longer"), 0o644))
		require.NoError(t, os.Chtimes(src, info.ModTime(), info.ModTime()))

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.True(t, got.Stale)
		assert.Equal(t, stale.ReasonSourceMetadataMismatch, got.Reason)
		assert.Equal(t, src, got.SourcePath)
	})

	t.Run("same second sha mismatch is stale", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		require.NoError(t, os.WriteFile(src, []byte("mode: one"), 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{
			MTime:   info.ModTime().Unix(),
			MTimeNS: info.ModTime().UnixNano(),
			Size:    info.Size(),
			SHA256:  sha256Hex([]byte("mode: one")),
		}
		require.NoError(t, os.WriteFile(src, []byte("mode: two"), 0o644))
		require.NoError(t, os.Chtimes(src, info.ModTime(), info.ModTime()))

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.True(t, got.Stale)
		assert.Equal(t, stale.ReasonSourceMetadataMismatch, got.Reason)
	})

	t.Run("newer nanosecond mtime is stale", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		require.NoError(t, os.WriteFile(src, []byte("mode: full"), 0o644))
		past := time.Unix(100, 100)
		now := time.Unix(100, 200)
		require.NoError(t, os.Chtimes(src, past, past))
		recorded := stale.SourceMetadata{MTime: past.Unix(), MTimeNS: past.UnixNano(), Size: int64(len("mode: full"))}
		require.NoError(t, os.Chtimes(src, now, now))

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.True(t, got.Stale)
		assert.Equal(t, stale.ReasonSourceNewer, got.Reason)
	})
}

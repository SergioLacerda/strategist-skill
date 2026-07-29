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

	t.Run("fresh strong source matching size mtime and sha is not stale", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		content := []byte("mode: same")
		require.NoError(t, os.WriteFile(src, content, 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{
			MTime:   info.ModTime().Unix(),
			MTimeNS: info.ModTime().UnixNano(),
			Size:    info.Size(),
			SHA256:  sha256Hex(content),
		}

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.False(t, got.Stale)
		assert.Equal(t, stale.ReasonFresh, got.Reason)
	})

	t.Run("strong source without recorded nanosecond mtime falls back to unix seconds", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		content := []byte("mode: full")
		require.NoError(t, os.WriteFile(src, content, 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{MTime: info.ModTime().Unix(), Size: info.Size(), SHA256: sha256Hex(content)}

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.False(t, got.Stale)
		assert.Equal(t, stale.ReasonFresh, got.Reason)
	})

	t.Run("missing strong source is stale", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		src := filepath.Join(dir, "gone.yaml")
		recorded := stale.SourceMetadata{MTime: time.Now().Unix(), Size: 10}

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.True(t, got.Stale)
		assert.Equal(t, stale.ReasonMissingSource, got.Reason)
		assert.Equal(t, src, got.SourcePath)
	})

	t.Run("strong source sha256 read error is treated as mismatch", func(t *testing.T) {
		t.Parallel()
		if os.Getuid() == 0 {
			t.Skip("permission tests do not apply when running as root")
		}
		dir := t.TempDir()
		src := filepath.Join(dir, "active.yaml")
		content := []byte("mode: secret")
		require.NoError(t, os.WriteFile(src, content, 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{
			MTime:   info.ModTime().Unix(),
			MTimeNS: info.ModTime().UnixNano(),
			Size:    info.Size(),
			SHA256:  sha256Hex(content),
		}
		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		require.NoError(t, os.Chmod(src, 0o000))
		t.Cleanup(func() { _ = os.Chmod(src, 0o644) })

		got, err := checker.Check(art)
		require.NoError(t, err)
		assert.True(t, got.Stale)
		assert.Equal(t, stale.ReasonSourceMetadataMismatch, got.Reason)
	})

	t.Run("strong source stat error surfaces", func(t *testing.T) {
		t.Parallel()
		if os.Getuid() == 0 {
			t.Skip("permission tests do not apply when running as root")
		}
		dir := t.TempDir()
		subdir := filepath.Join(dir, "locked")
		require.NoError(t, os.Mkdir(subdir, 0o755))
		src := filepath.Join(subdir, "active.yaml")
		require.NoError(t, os.WriteFile(src, []byte("data"), 0o644))
		info, err := os.Stat(src)
		require.NoError(t, err)
		recorded := stale.SourceMetadata{MTime: info.ModTime().Unix(), MTimeNS: info.ModTime().UnixNano(), Size: info.Size()}

		art := writeArtifactWithStrongSource(t, dir, src, recorded)
		require.NoError(t, os.Chmod(subdir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })

		_, err = checker.Check(art)
		require.Error(t, err)
	})
}

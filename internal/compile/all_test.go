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

func TestCompileAll(t *testing.T) {
	t.Parallel()
	t.Run("produces all four artifacts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		c := compile.Compiler{}
		require.NoError(t, c.CompileAll(dir, kiPath))

		compiledDir := filepath.Join(dir, ".compiled")
		for _, name := range []string{".config.gz", ".domain.gz", ".index.gz", ".manifest.gz"} {
			assert.FileExists(t, filepath.Join(compiledDir, name))
		}
	})

	t.Run("manifest contains sha256 for all artifacts", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		require.NoError(t, compile.Compiler{}.CompileAll(dir, kiPath))

		var manifest map[string]any
		testutil.ReadGzJSON(t, filepath.Join(dir, ".compiled", ".manifest.gz"), &manifest)
		artifacts := manifest["artifacts"].(map[string]any)
		for _, name := range []string{".config.gz", ".domain.gz", ".index.gz"} {
			sha, ok := artifacts[name].(string)
			require.True(t, ok, "artifact %s missing from manifest", name)
			assert.Contains(t, sha, "sha256:", "artifact %s should have sha256 prefix", name)
		}
	})

	t.Run("fails if active.yaml missing — manifest not written", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: []\nload_by_task_type: {}\n"),
			0o644,
		))
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		err := compile.Compiler{}.CompileAll(dir, kiPath)
		require.Error(t, err)
		assert.NoFileExists(t, filepath.Join(dir, ".compiled", ".manifest.gz"))
	})

	t.Run("fails if knowledge index missing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		err := compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "nonexistent.yaml"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "index")
	})

	t.Run("fails if index.yaml missing (domain step)", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		kiPath := filepath.Join(dir, "knowledge.index.yaml")
		require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))

		err := compile.Compiler{}.CompileAll(dir, kiPath)
		require.Error(t, err)
		assert.ErrorContains(t, err, "domain")
	})

	t.Run("partial failure after prior success removes stale manifest", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)
		kiPath := filepath.Join(dir, "knowledge.index.yaml")

		c := compile.Compiler{}
		require.NoError(t, c.CompileAll(dir, kiPath))
		manifestPath := filepath.Join(dir, ".compiled", ".manifest.gz")
		require.FileExists(t, manifestPath)

		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "index.yaml"),
			[]byte("load_always: [unterminated\n"),
			0o644,
		))

		err := c.CompileAll(dir, kiPath)
		require.Error(t, err)
		assert.NoFileExists(t, manifestPath, "manifest must be removed after a partial failure so absence is the sole staleness signal")
	})
}

package embed_test

import (
	"os"
	"path/filepath"
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_Extract_ReadOnlyTarget(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	// Make target directory read-only so MkdirAll / WriteFile inside it fails
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := embedpkg.Extractor{}.Extract(dir)
	require.Error(t, err)
}

func TestExtractor_Extract(t *testing.T) {
	t.Run("extracts defaults into target directory", func(t *testing.T) {
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir))

		// Core files that must always be extracted
		expectedFiles := []string{
			"SKILL.md",
			"knowledge.index.yaml",
			"skill.yaml",
			"index.yaml",
		}
		for _, f := range expectedFiles {
			assert.FileExists(t, filepath.Join(dir, f), "expected embedded file: %s", f)
		}

		// Core directories
		expectedDirs := []string{
			"personas",
			"roles",
			"schemas",
			"templates",
		}
		for _, d := range expectedDirs {
			assert.DirExists(t, filepath.Join(dir, d), "expected embedded dir: %s", d)
		}
	})

	t.Run("extract is idempotent", func(t *testing.T) {
		dir := t.TempDir()
		ext := embedpkg.Extractor{}
		require.NoError(t, ext.Extract(dir))
		require.NoError(t, ext.Extract(dir), "second extract should not fail")
	})

	t.Run("extracted SKILL.md is non-empty", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, embedpkg.Extractor{}.Extract(dir))
		data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

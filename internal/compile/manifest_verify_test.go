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

func TestVerifyManifest_NoDrift(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	require.NoError(t, compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "knowledge.index.yaml")))

	drift, err := compile.VerifyManifest(filepath.Join(dir, ".compiled"))
	require.NoError(t, err)
	assert.Empty(t, drift)
}

func TestVerifyManifest_HashMismatch(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	require.NoError(t, compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "knowledge.index.yaml")))

	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.WriteFile(filepath.Join(compiledDir, ".config.gz"), []byte("tampered"), 0o644))

	drift, err := compile.VerifyManifest(compiledDir)
	require.NoError(t, err)
	require.Len(t, drift, 1)
	assert.Contains(t, drift[0], ".config.gz")
	assert.Contains(t, drift[0], "hash mismatch")
}

func TestVerifyManifest_MissingArtifact(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	require.NoError(t, compile.Compiler{}.CompileAll(dir, filepath.Join(dir, "knowledge.index.yaml")))

	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.Remove(filepath.Join(compiledDir, ".domain.gz")))

	drift, err := compile.VerifyManifest(compiledDir)
	require.NoError(t, err)
	require.Len(t, drift, 1)
	assert.Contains(t, drift[0], ".domain.gz")
	assert.Contains(t, drift[0], "missing")
}

func TestVerifyManifest_NoManifest(t *testing.T) {
	compiledDir := t.TempDir()
	drift, err := compile.VerifyManifest(compiledDir)
	require.NoError(t, err)
	require.Len(t, drift, 1)
	assert.Contains(t, drift[0], "not found")
}

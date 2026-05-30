//go:build integration

package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	out := filepath.Join(dir, ".compiled", ".config.gz")

	require.NoError(t, compile.Config(dir, out))
	require.FileExists(t, out)

	var artifact map[string]interface{}
	testutil.ReadGzJSON(t, out, &artifact)

	assert.Equal(t, "strategist-compiled-config/1.0", artifact["schema"])
	assert.NotNil(t, artifact["compiled_at"])
	assert.NotNil(t, artifact["sources"])
	assert.NotNil(t, artifact["active"])
	assert.NotNil(t, artifact["personas"])
	assert.NotNil(t, artifact["roles"])
}

func TestCompileDomain_EmptyIndex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	indexYAML := "load_always: []\nload_by_task_type: {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(indexYAML), 0o644))
	out := filepath.Join(dir, ".compiled", ".domain.gz")

	require.NoError(t, compile.Domain(dir, out))
	require.FileExists(t, out)

	var artifact map[string]interface{}
	testutil.ReadGzJSON(t, out, &artifact)

	assert.Equal(t, "strategist-compiled-domain/1.0", artifact["schema"])
	assert.NotNil(t, artifact["load_always"])
	assert.NotNil(t, artifact["load_by_task_type"])
}

func TestCompileIndex_EmptySources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))
	out := filepath.Join(dir, ".compiled", ".index.gz")

	require.NoError(t, compile.Index(kiPath, out))
	require.FileExists(t, out)

	var artifact map[string]interface{}
	testutil.ReadGzJSON(t, out, &artifact)

	assert.Equal(t, "strategist-compiled-index/1.0", artifact["schema"])
	assert.NotNil(t, artifact["tags"])
}

func TestCompileAll_ProducesAllArtifacts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	// domain index
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
		assert.FileExists(t, filepath.Join(compiledDir, name), "expected artifact %s", name)
	}

	// Manifest must reference all three artifact checksums
	var manifest map[string]interface{}
	testutil.ReadGzJSON(t, filepath.Join(compiledDir, ".manifest.gz"), &manifest)
	artifacts, ok := manifest["artifacts"].(map[string]interface{})
	require.True(t, ok, "manifest.artifacts must be an object")
	assert.Contains(t, artifacts, ".config.gz")
	assert.Contains(t, artifacts, ".domain.gz")
	assert.Contains(t, artifacts, ".index.gz")
}

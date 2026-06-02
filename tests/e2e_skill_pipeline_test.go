//go:build integration

package tests_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDir(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Dir(file)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	return filepath.Clean(filepath.Join(testDir(t), ".."))
}

func prepareE2ERoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	extractor := embedpkg.Extractor{}
	require.NoError(t, extractor.Extract(root, false))

	activeSource := filepath.Join(repoRoot(t), ".strategist", "active.yaml")
	activeData, err := os.ReadFile(activeSource)
	require.NoError(t, err, "read %s", activeSource)
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), activeData, 0o644))

	return root
}

func TestE2E_EmbedExtractCompile_ProducesBootstrapReadyConfig(t *testing.T) {
	t.Parallel()

	root := prepareE2ERoot(t)
	kiPath := filepath.Join(root, "knowledge.index.yaml")
	require.NoError(t, compile.Compiler{}.CompileAll(root, kiPath))

	compiledConfigPath := filepath.Join(root, ".compiled", ".config.gz")
	require.FileExists(t, compiledConfigPath)

	var artifact map[string]any
	testutil.ReadGzJSON(t, compiledConfigPath, &artifact)

	active, ok := artifact["active"].(map[string]any)
	require.True(t, ok, "compiled active must be an object")

	slots, ok := active["slots"].(map[string]any)
	require.True(t, ok, "active.slots must be an object")
	for _, key := range []string{"discovery", "refinement", "execution"} {
		assert.Contains(t, slots, key, "active.slots must define %s", key)
	}

	language, ok := active["language"].(map[string]any)
	require.True(t, ok, "active.language must be an object")
	assert.Equal(t, "pt-BR", language["chat"])
	assert.NotEmpty(t, active["base_path"], "active.base_path must be populated")

	personas, ok := artifact["personas"].(map[string]any)
	require.True(t, ok, "compiled personas must be an object")

	epic, ok := personas["epic"].(map[string]any)
	require.True(t, ok, "personas.epic must be an object")

	contentByLang, ok := epic["content_by_lang"].(map[string]any)
	require.True(t, ok, "personas.epic.content_by_lang must be an object")
	assert.Contains(t, contentByLang, "pt-BR")
	assert.Contains(t, contentByLang, "en")
}

func TestE2E_EmbedExtractCompile_DomainHasLoadAlways(t *testing.T) {
	t.Parallel()

	root := prepareE2ERoot(t)
	kiPath := filepath.Join(root, "knowledge.index.yaml")
	require.NoError(t, compile.Compiler{}.CompileAll(root, kiPath))

	var artifact map[string]any
	testutil.ReadGzJSON(t, filepath.Join(root, ".compiled", ".domain.gz"), &artifact)

	loadAlways, ok := artifact["load_always"].(map[string]any)
	require.True(t, ok, "compiled load_always must be an object")
	require.NotEmpty(t, loadAlways, "load_always must not be empty")
}

func TestE2E_StaleCheck_FreshAfterCompile(t *testing.T) {
	t.Parallel()

	root := prepareE2ERoot(t)
	kiPath := filepath.Join(root, "knowledge.index.yaml")
	require.NoError(t, compile.Compiler{}.CompileAll(root, kiPath))

	artifactPath := filepath.Join(root, ".compiled", ".config.gz")
	fresh, err := stale.Checker{}.IsStale(artifactPath)
	require.NoError(t, err)
	assert.False(t, fresh, "compiled artifact must be fresh immediately after compile")
}

func TestE2E_StaleCheck_StaleWhenAbsent(t *testing.T) {
	t.Parallel()

	artifactPath := filepath.Join(t.TempDir(), ".compiled", ".config.gz")
	staleArtifact, err := stale.Checker{}.IsStale(artifactPath)
	require.NoError(t, err)
	assert.True(t, staleArtifact, "missing compiled artifact must be stale")
}

func TestE2E_CompiledManifest_ReferencesAllArtifacts(t *testing.T) {
	t.Parallel()

	root := prepareE2ERoot(t)
	kiPath := filepath.Join(root, "knowledge.index.yaml")
	require.NoError(t, compile.Compiler{}.CompileAll(root, kiPath))

	var manifest map[string]any
	testutil.ReadGzJSON(t, filepath.Join(root, ".compiled", ".manifest.gz"), &manifest)

	artifacts, ok := manifest["artifacts"].(map[string]any)
	require.True(t, ok, "manifest.artifacts must be an object")
	for _, name := range []string{".config.gz", ".domain.gz", ".index.gz"} {
		assert.Contains(t, artifacts, name, "manifest must reference %s", name)
	}
}

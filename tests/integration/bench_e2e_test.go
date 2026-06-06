//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

// BenchmarkInstallAndCompile_RealEmbed measures CompileAll against the full
// embedded defaults — 20+ YAML files — giving production-representative numbers.
// Each iteration starts with a cold .compiled/ directory.
func BenchmarkInstallAndCompile_RealEmbed(b *testing.B) {
	root := b.TempDir()
	extractor := embedpkg.Extractor{}
	require.NoError(b, extractor.Extract(root, false))

	// Provide active.yaml from the embedded standalone template.
	activeData, err := extractor.ReadFile("templates/epic-standalone.yaml")
	require.NoError(b, err)
	require.NoError(b, os.WriteFile(filepath.Join(root, "active.yaml"), activeData, 0o644))

	kiPath := filepath.Join(root, "knowledge.index.yaml")
	compiledDir := filepath.Join(root, ".compiled")

	b.ResetTimer()
	for range b.N {
		require.NoError(b, compile.Compiler{}.CompileAll(root, kiPath))
		_ = os.RemoveAll(compiledDir)
	}
}

// BenchmarkStaleCheck_CacheMiss measures IsStale when the artifact does not
// exist — the path taken on a brand-new install before any compilation.
func BenchmarkStaleCheck_CacheMiss(b *testing.B) {
	dir := b.TempDir()
	// Populate source files so the checker has real mtimes to compare.
	testutil.MinimalRoot(b, dir)

	absent := filepath.Join(dir, ".compiled", ".config.gz")
	checker := stale.Checker{}

	b.ResetTimer()
	for range b.N {
		_, _ = checker.IsStale(absent)
	}
}

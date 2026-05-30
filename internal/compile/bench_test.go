package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

func BenchmarkCompileConfig(b *testing.B) {
	dir := b.TempDir()
	testutil.MinimalRoot(b, dir)
	out := filepath.Join(dir, ".compiled", ".config.gz")
	b.ResetTimer()
	for range b.N {
		require.NoError(b, compile.Config(dir, out))
	}
}

func BenchmarkCompileAll(b *testing.B) {
	dir := b.TempDir()
	testutil.MinimalRoot(b, dir)
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	b.ResetTimer()
	for range b.N {
		require.NoError(b, compile.Compiler{}.CompileAll(dir, kiPath))
		// Remove compiled dir so each iteration starts fresh.
		_ = os.RemoveAll(filepath.Join(dir, ".compiled"))
	}
}

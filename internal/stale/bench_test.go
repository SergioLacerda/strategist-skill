package stale_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
)

func BenchmarkIsStale_Fresh(b *testing.B) {
	dir := b.TempDir()
	art := filepath.Join(dir, ".config.gz")
	testutil.WriteGzJSON(b, art, map[string]any{"sources": map[string]int64{}})
	testutil.WriteGzJSON(b, filepath.Join(dir, ".manifest.gz"), map[string]any{})
	checker := stale.Checker{}
	b.ResetTimer()
	for range b.N {
		_, _ = checker.IsStale(art)
	}
}

func BenchmarkIsStale_Stale(b *testing.B) {
	dir := b.TempDir()
	src := filepath.Join(dir, "active.yaml")
	if err := os.WriteFile(src, []byte("mode: full"), 0o644); err != nil {
		b.Fatal(err)
	}
	past := time.Now().Add(-1 * time.Hour)
	art := filepath.Join(dir, ".config.gz")
	testutil.WriteGzJSON(b, art, map[string]any{
		"sources": map[string]int64{src: past.Unix()},
	})
	testutil.WriteGzJSON(b, filepath.Join(dir, ".manifest.gz"), map[string]any{})
	checker := stale.Checker{}
	b.ResetTimer()
	for range b.N {
		_, _ = checker.IsStale(art)
	}
}

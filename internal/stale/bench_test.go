package stale_test

import (
	"fmt"
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

func BenchmarkCheck_ManyStrongSources(b *testing.B) {
	dir := b.TempDir()
	sourceStats, legacySources := buildStrongSourceSet(b, dir, 1000)

	art := filepath.Join(dir, ".config.gz")
	testutil.WriteGzJSON(b, art, map[string]any{
		"source_stats": sourceStats,
		"sources":      legacySources,
	})
	testutil.WriteGzJSON(b, filepath.Join(dir, ".manifest.gz"), map[string]any{})

	checker := stale.Checker{}
	b.ResetTimer()
	for range b.N {
		_, _ = checker.Check(art)
	}
}

func buildStrongSourceSet(b *testing.B, dir string, count int) (map[string]stale.SourceMetadata, map[string]int64) {
	b.Helper()
	sourceStats := make(map[string]stale.SourceMetadata, count)
	legacySources := make(map[string]int64, count)
	for i := range count {
		path := filepath.Join(dir, "sources", fmt.Sprintf("source-%04d.yaml", i))
		sourceStats[path], legacySources[path] = writeStrongSource(b, path)
	}
	return sourceStats, legacySources
}

func writeStrongSource(b *testing.B, path string) (stale.SourceMetadata, int64) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("key: value\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		b.Fatal(err)
	}
	return stale.SourceMetadata{
		MTime:   info.ModTime().Unix(),
		MTimeNS: info.ModTime().UnixNano(),
		Size:    info.Size(),
	}, info.ModTime().Unix()
}

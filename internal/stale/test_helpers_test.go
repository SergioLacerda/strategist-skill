package stale_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/require"
)

func writeManifestForArtifact(t testing.TB, artifactPath string) {
	t.Helper()
	data, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	testutil.WriteGzJSON(t, filepath.Join(filepath.Dir(artifactPath), ".manifest.gz"), map[string]any{
		"artifacts": map[string]string{
			filepath.Base(artifactPath): "sha256:" + sha256Hex(data),
		},
	})
}

func writeArtifactWithStrongSource(t *testing.T, dir, src string, meta stale.SourceMetadata) string {
	t.Helper()
	art := filepath.Join(dir, ".config.gz")
	testutil.WriteGzJSON(t, art, map[string]any{
		"source_stats": map[string]stale.SourceMetadata{src: meta},
		"sources":      map[string]int64{src: meta.MTime},
	})
	writeManifestForArtifact(t, art)
	return art
}

func sha256Hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

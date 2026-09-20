package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func writePackage(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["SKILL.md"] = "---\nname: demo\nmetadata:\n  version: \"1.0\"\n---\nbody\n"
	for rel, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	}
	return dir
}

func big(n int) string { return strings.Repeat("x", n) }

func TestResolveEmbeddedPackageAllowsLargeFilesOnlyUnderRuntime(t *testing.T) {
	over := domain.MaxPluginManifestBytes + 1
	dir := writePackage(t, map[string]string{"runtime/dist/bundle.mjs": big(over)})

	_, err := ResolveLocalPackage(dir)
	require.ErrorContains(t, err, "exceeds", "custom packages keep the strict limit")

	pkg, err := ResolveEmbeddedPackage(dir)
	require.NoError(t, err)
	require.Contains(t, pkg.Digest, "sha256:")
	require.Greater(t, pkg.ArtifactSize, int64(over))
}

func TestResolveEmbeddedPackageStillLimitsEverythingElse(t *testing.T) {
	over := domain.MaxPluginManifestBytes + 1
	outside := writePackage(t, map[string]string{"references/huge.md": big(over)})
	_, err := ResolveEmbeddedPackage(outside)
	require.ErrorContains(t, err, "exceeds")

	tooBig := writePackage(t, map[string]string{"runtime/bundle.mjs": big(MaxEmbeddedRuntimeFileBytes + 1)})
	_, err = ResolveEmbeddedPackage(tooBig)
	require.ErrorContains(t, err, "exceeds")
}

func TestResolveEmbeddedPackageDigestMatchesContentAndIsStable(t *testing.T) {
	files := map[string]string{"runtime/a.mjs": big(domain.MaxPluginManifestBytes + 10)}
	first, err := ResolveEmbeddedPackage(writePackage(t, files))
	require.NoError(t, err)
	second, err := ResolveEmbeddedPackage(writePackage(t, files))
	require.NoError(t, err)
	require.Equal(t, first.Digest, second.Digest)

	files["runtime/a.mjs"] += "y"
	changed, err := ResolveEmbeddedPackage(writePackage(t, files))
	require.NoError(t, err)
	require.NotEqual(t, first.Digest, changed.Digest)
}

package check

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStrictChecks_ManifestVerifyError(t *testing.T) {
	dir := t.TempDir()
	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(compiledDir, ".manifest.gz"), []byte("not gzip"), 0o644))

	errs := runStrictChecks(dir)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "strict: verify manifest") {
			found = true
		}
	}
	assert.True(t, found, "expected a 'strict: verify manifest' error, got: %v", errs)
}

func TestValidateRuntimeDefaultFile_ReadPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	rel := "SKILL.md"
	path := filepath.Join(dir, rel)
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	msg, ok := validateRuntimeDefaultFile(dir, rel, embedpkg.Extractor{}, domain.InstallManifest{}, false, nil)
	require.True(t, ok)
	assert.Contains(t, msg, "runtime_stale: read")
}

func TestValidateRuntimeDefaultFile_EmbeddedUnreadable(t *testing.T) {
	dir := t.TempDir()
	rel := "not-a-real-embedded-path.yaml"
	path := filepath.Join(dir, rel)
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o644))

	msg, ok := validateRuntimeDefaultFile(dir, rel, embedpkg.Extractor{}, domain.InstallManifest{}, false, nil)
	require.True(t, ok)
	assert.Contains(t, msg, "embedded default")
	assert.Contains(t, msg, "unreadable")
}

func TestClassifyRuntimeStale_UnknownManifestFile(t *testing.T) {
	manifest := domain.InstallManifest{
		Schema:    "strategist.install-manifest.v1",
		PackageID: "test",
		Files:     []domain.InstallManifestFile{{Path: "other.yaml", Owner: domain.RuntimeFileNormative, SHA256: "abc"}},
	}
	decision := classifyRuntimeStale([]byte("content"), "SKILL.md", manifest, true, nil)
	assert.Equal(t, domain.RuntimeDecisionUnknownManifest, decision)
}

func TestReadInstallManifest_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, domain.InstallManifestRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	_, loaded, err := readInstallManifest(dir)
	require.Error(t, err)
	assert.False(t, loaded)
	assert.Contains(t, err.Error(), "read install manifest")
}

func TestReadInstallManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, domain.InstallManifestRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, loaded, err := readInstallManifest(dir)
	require.Error(t, err)
	assert.False(t, loaded)
	assert.Contains(t, err.Error(), "parse install manifest")
}

func TestValidateRuntimeDefaultParity_ManifestUnreadableStillChecksFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, domain.InstallManifestRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(manifestPath), 0o755))
	require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(manifestPath, 0o644) })
	if os.Getuid() == 0 {
		t.Skip("running as root — file permission checks do not apply")
	}

	errs := validateRuntimeDefaultParity(dir)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "install manifest unreadable") {
			found = true
		}
	}
	assert.True(t, found, "expected an 'install manifest unreadable' error, got: %v", errs)
}

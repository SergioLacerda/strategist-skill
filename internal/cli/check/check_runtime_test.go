package check

import (
	"encoding/json"
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
	decision := classifyRuntimeStale([]byte("content"), []byte("embedded"), "SKILL.md", manifest, true, nil)
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

// A check run by a binary older than the runtime must not advise reinstalling,
// which would downgrade the runtime; it names the stale binary instead.
func TestClassifyRuntimeStale_BinaryOlderThanRuntime(t *testing.T) {
	newer, older := []byte("newer default"), []byte("older default")
	manifest := domain.InstallManifest{Files: []domain.InstallManifestFile{{
		Path: "SKILL.md", Owner: domain.RuntimeFileNormative,
		SHA256: domain.SHA256Hex(newer), History: []string{domain.SHA256Hex(older)},
	}}}
	decision := classifyRuntimeStale(newer, older, "SKILL.md", manifest, true, nil)
	assert.Equal(t, domain.RuntimeDecisionDowngrade, decision)
	assert.Contains(t, domain.FormatRuntimeStaleDiagnostic("SKILL.md", decision), "make install")

	assert.Equal(t, domain.RuntimeDecisionAutoUpgrade, classifyRuntimeStale(newer, []byte("brand new"), "SKILL.md", manifest, true, nil))
}

func TestValidateRuntimeDefaultParity_CompleteRuntimeReportsNothing(t *testing.T) {
	dir := t.TempDir()
	writeNormativeRuntimeFiles(t, dir)

	assert.Empty(t, validateRuntimeDefaultParity(dir))
}

func TestValidateRuntimeDefaultParity_ReportsEveryMissingRequiredFile(t *testing.T) {
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		t.Run(file.Path, func(t *testing.T) {
			dir := t.TempDir()
			writeNormativeRuntimeFiles(t, dir)
			require.NoError(t, os.Remove(filepath.Join(dir, filepath.FromSlash(file.Path))))

			errs := validateRuntimeDefaultParity(dir)

			assert.Equal(t, []string{domain.FormatRuntimeMissingDiagnostic(file.Path)}, errs)
		})
	}
}

func TestValidateRuntimeDefaultParity_ReportsAMissingGeneratedFile(t *testing.T) {
	for _, rel := range domain.GeneratedRuntimeFilePaths() {
		t.Run(rel, func(t *testing.T) {
			dir := t.TempDir()
			writeNormativeRuntimeFiles(t, dir)
			require.NoError(t, os.Remove(filepath.Join(dir, filepath.FromSlash(rel))))

			errs := validateRuntimeDefaultParity(dir)

			assert.Equal(t, []string{domain.FormatGeneratedRuntimeMissingDiagnostic(rel)}, errs)
		})
	}
}

func TestValidateRuntimeDefaultParity_DriftStillReportedNotAsMissing(t *testing.T) {
	dir := t.TempDir()
	writeNormativeRuntimeFiles(t, dir)
	path := filepath.Join(dir, "SKILL.md")
	require.NoError(t, os.WriteFile(path, []byte("edited\n"), 0o644))

	errs := validateRuntimeDefaultParity(dir)

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "runtime_stale")
	assert.NotContains(t, errs[0], "runtime_missing")
}

func TestCheckCmd_JSON_BlockedWhenANormativeFileIsDeleted(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "SKILL.md")))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		require.Error(t, checkCmd.RunE(checkCmd, nil))
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	assert.Contains(t, strings.Join(result.Warnings, "\n"), domain.FormatRuntimeMissingDiagnostic("SKILL.md"))
}

func TestCheckCmd_JSON_BlockedWhenTheGeneratedAgentProtocolIsDeleted(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "agent-protocol.md")))
	checkRoot = dir
	checkJSON = true

	out := captureStdout(t, func() {
		require.Error(t, checkCmd.RunE(checkCmd, nil))
	})

	var result domain.PreflightResult
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "blocked", result.Status)
	assert.Contains(t, strings.Join(result.Warnings, "\n"), domain.FormatGeneratedRuntimeMissingDiagnostic("agent-protocol.md"))
}

func TestCheckCmd_StrictReportsAMissingNormativeFile(t *testing.T) {
	resetCheckFlags(t)
	dir := minimalCheckRoot(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "protocol.md")))
	checkRoot = dir
	checkStrict = true

	errOut := captureStderr(t, func() {
		require.Error(t, checkCmd.RunE(checkCmd, nil))
	})

	assert.Contains(t, errOut, domain.FormatRuntimeMissingDiagnostic("protocol.md"))
}

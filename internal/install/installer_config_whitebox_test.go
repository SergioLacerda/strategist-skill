package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyConfig_PreservesExistingActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	customContent := "mode: epic\nbase_path: .custom\n"
	activeYAMLPath := filepath.Join(dir, ".strategist", "active.yaml")
	require.NoError(t, os.WriteFile(activeYAMLPath, []byte(customContent), 0o644))
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	got, err := os.ReadFile(activeYAMLPath)
	require.NoError(t, err)
	assert.Equal(t, customContent, string(got), "re-install must not overwrite user-customized active.yaml")
}

func TestApplyConfig_ForceOverwritesActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))

	activeYAMLPath := filepath.Join(dir, ".strategist", "active.yaml")
	require.NoError(t, os.WriteFile(activeYAMLPath, []byte("mode: epic\nbase_path: .custom\n"), 0o644))
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true}))

	got, err := os.ReadFile(activeYAMLPath)
	require.NoError(t, err)
	assert.Equal(t, "mode: epic\nbase_path: .analysis\n", string(got), "--force must overwrite active.yaml with embedded template")
}

func TestApplyConfig_ReadFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{Extractor: templateReadErrorExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	err := svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.Error(t, err)
	assert.ErrorContains(t, err, "read template")
}

type templateReadErrorExtractor struct {
	minimalExtractor
}

func (e templateReadErrorExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == "templates/epic-standalone.yaml" {
		return nil, fmt.Errorf("templateReadErrorExtractor: read error for %s", relPath)
	}
	return e.minimalExtractor.ReadFile(relPath)
}

func TestEnsureGitignore_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte("*.log"), 0o644))
	require.NoError(t, ensureGitignore(dir))
	data, err := os.ReadFile(gi)
	require.NoError(t, err)
	assert.Contains(t, string(data), ".strategist/.compiled/")
}

func TestEnsureGitignore_AlreadyPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte(".strategist/.compiled/\n"), 0o644))
	require.NoError(t, ensureGitignore(dir))
	data, err := os.ReadFile(gi)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(data), ".strategist/.compiled/"))
}

func TestEnsureGitignore_OpenError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := ensureGitignore(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "open .gitignore")
}

func TestEnsureGitignore_ReadError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	gi := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(gi, []byte(""), 0o000))
	t.Cleanup(func() { _ = os.Chmod(gi, 0o644) })
	err := ensureGitignore(dir)
	require.Error(t, err)
}

func TestWriteActiveYAML_ReadOnlyDir(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	err := writeActiveYAML(dir, domain.WizardConfig{
		Mode: "pragmatic", BasePath: ".", UILanguage: "pt", DocLanguage: "pt", ChatLanguage: "pt", CodeLanguage: "pt",
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-explore", ExecutionProvider: "sniper",
	})
	require.Error(t, err)
}

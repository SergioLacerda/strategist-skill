package install_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_Silent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{}
	comp := &mockCompiler{}

	svc := newSvc(t, ext, comp)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
	}))

	strategistDir := filepath.Join(dir, ".strategist")
	assert.Equal(t, strategistDir, ext.calledPaths[0])
	assert.True(t, comp.called, "compiler must be called after install")

	for _, f := range []string{"active.yaml", "SKILL.md", "knowledge.index.yaml"} {
		assert.FileExists(t, filepath.Join(strategistDir, f))
	}
	for _, d := range []string{"personas", "roles", "memory"} {
		assert.DirExists(t, filepath.Join(strategistDir, d))
	}

	shimPath := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	shimData, err := os.ReadFile(shimPath)
	require.NoError(t, err)
	shimStr := string(shimData)
	assert.Contains(t, shimStr, "name: strategist", "shim must have frontmatter")
	assert.Contains(t, shimStr, "skill_root: "+filepath.Join(dir, ".strategist"), "shim must pin project-local skill_root")
	assert.Contains(t, shimStr, "# SKILL", "shim must contain SKILL.md content from extractor")
}

func TestInstall_EnsuresGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, newSvc(t, &mockExtractor{}, &mockCompiler{}).Install(
		context.Background(), domain.InstallConfig{Target: dir, Silent: true},
	))
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(data), ".strategist/.compiled/")
}

func TestInstall_GlobalMode_DoesNotWriteGitignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
		Global: true,
	}))
	_, err := os.Stat(filepath.Join(dir, ".gitignore"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestInstall_GitignoreIdempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	cfg := domain.InstallConfig{Target: dir, Silent: true}

	require.NoError(t, svc.Install(context.Background(), cfg))
	require.NoError(t, svc.Install(context.Background(), cfg))

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	count := 0
	for _, line := range filepath.SplitList(string(data)) {
		if line == ".strategist/.compiled/" {
			count++
		}
	}
	assert.LessOrEqual(t, count, 2, "gitignore entry should not duplicate excessively")
}

func TestInstall_ExtractorFailurePropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{failWith: os.ErrPermission}
	err := newSvc(t, ext, &mockCompiler{}).Install(context.Background(), domain.InstallConfig{Target: dir})
	require.Error(t, err)
	assert.ErrorContains(t, err, "extract defaults")
}

func TestInstall_CompileFailureIsNonFatal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err, "compile failure must be non-fatal")
}

func TestInstall_NewInstaller(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	inst := install.NewInstaller(&mockExtractor{}, &mockCompiler{})
	err := inst.Install(domain.InstallConfig{Target: dir, Silent: true})
	require.NoError(t, err)
}

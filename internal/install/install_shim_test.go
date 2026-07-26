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

func TestInstall_NoShim_SkipsShimWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, NoShim: true,
	}))
	shimPath := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	_, err := os.Stat(shimPath)
	assert.ErrorIs(t, err, os.ErrNotExist, "--no-shim must not write under the home shim path")
}

func TestInstall_ShimPath_WritesToCustomPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customShim := filepath.Join(t.TempDir(), "custom", "SKILL.md")
	svc := newSvc(t, &mockExtractor{}, &mockCompiler{})
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, ShimPath: customShim,
	}))
	assert.FileExists(t, customShim)

	defaultShim := filepath.Join(svc.ShimHomeDir, ".claude", "skills", "strategist", "SKILL.md")
	_, err := os.Stat(defaultShim)
	assert.ErrorIs(t, err, os.ErrNotExist, "--shim-path must not also write the default home shim path")
}

func TestInstall_ShimPath_RollsBackOnLaterFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	customShim := filepath.Join(t.TempDir(), "custom", "SKILL.md")
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, ShimPath: customShim, StrictCompile: true,
	})
	require.Error(t, err)
	_, statErr := os.Stat(customShim)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "custom shim path must be rolled back on later fatal failure")
}

func TestInstall_DoesNotPopulateGlobalRuntimeByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := &mockExtractor{}
	comp := &mockCompiler{}
	svc := newSvc(t, ext, comp)

	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{
		Target: dir,
		Silent: true,
	}))

	globalDir := filepath.Join(svc.ShimHomeDir, ".strategist")
	_, statErr := os.Stat(globalDir)
	require.ErrorIs(t, statErr, os.ErrNotExist, "default install must not create global runtime dir")
	assert.Len(t, ext.calledPaths, 1, "extractor must be called only once for local install")
}

func TestInstall_PreexistingGlobalDirUntouched(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	shimHome := t.TempDir()

	globalDir := filepath.Join(shimHome, ".strategist")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(globalDir, "SENTINEL"), []byte("keep"), 0o644))

	svc := install.Service{Extractor: &mockExtractor{}, Compiler: &mockCompiler{}, ShimHomeDir: shimHome}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	assert.DirExists(t, globalDir, "pre-existing global dir must remain untouched")
	assert.FileExists(t, filepath.Join(globalDir, "SENTINEL"), "global dir contents must be preserved")
}

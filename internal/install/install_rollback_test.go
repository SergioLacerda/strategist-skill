package install_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_StrictCompileMakesFailureFatal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, StrictCompile: true,
	})
	require.Error(t, err, "strict-compile install must fail on a CompileAll error")
	require.ErrorContains(t, err, "strict compile")

	strategistDir := filepath.Join(dir, ".strategist")
	_, statErr := os.Stat(strategistDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "fresh install must be fully rolled back on strict-compile failure")
}

func TestInstall_RollbackRemovesFullTreeOnFreshInstallFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, StrictCompile: true,
	})
	require.Error(t, err)

	strategistDir := filepath.Join(dir, ".strategist")
	_, statErr := os.Stat(strategistDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist, ".strategist/ must be fully removed, not left partially extracted")
}

func TestInstall_RollbackPreservesPreexistingStrategistDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	strategistDir := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "SENTINEL"), []byte("keep"), 0o644))

	comp := &mockCompiler{failErr: os.ErrNotExist}
	err := newSvc(t, &mockExtractor{}, comp).Install(context.Background(), domain.InstallConfig{
		Target: dir, Silent: true, Force: true, StrictCompile: true,
	})
	require.Error(t, err)
	assert.FileExists(t, filepath.Join(strategistDir, "SENTINEL"), "reinstall over an existing tree must never delete pre-existing content on rollback")
}

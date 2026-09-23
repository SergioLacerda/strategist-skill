package cliutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadActiveConfigReadsTheLevelingBlock(t *testing.T) {
	dir := t.TempDir()
	body := "mode: epic\nbase_path: .analysis\nleveling:\n  mode: manual\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(body), 0o600))

	cfg, err := cliutil.LoadActiveConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, ".analysis", cfg.BasePath)
	assert.Equal(t, domain.LevelingModeManual, cfg.Leveling.EffectiveMode())
	assert.True(t, cfg.Leveling.HostPassthrough())
}

func TestLoadActiveConfigMissingFileIsErrNotExist(t *testing.T) {
	_, err := cliutil.LoadActiveConfig(t.TempDir())
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist, "callers can tell a missing file from a broken one")
}

func TestLoadActiveConfigRejectsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(":\n - [broken"), 0o600))
	_, err := cliutil.LoadActiveConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse active.yaml")
}

func TestResolveActiveBasePathUsesTheSharedLoader(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\n"), 0o600))
	_, base, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(filepath.Dir(root), ".analysis"), base)
}

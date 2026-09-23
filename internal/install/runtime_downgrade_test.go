package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const downgradePath = "contracts/machine/preflight.yaml"

func installWith(t *testing.T, dir string, ext runtimeDefaultsExtractor, cfg domain.InstallConfig) error {
	t.Helper()
	cfg.Target, cfg.Silent, cfg.NoShim = dir, true, true
	return runtimeDefaultService(ext).Install(context.Background(), cfg)
}

// v1 is installed, then v2 by a newer binary; an older binary carrying v1
// must not silently roll the runtime back (the 2026-09-22 regression, where
// ~/.local/bin/strategist reverted freshly shipped contracts).
func TestInstall_RefusesADowngradeFromAnOlderBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	older := newRuntimeDefaultsExtractor(nil)
	newer := newRuntimeDefaultsExtractor(map[string]string{downgradePath: downgradePath + " v2\n"})
	require.NoError(t, installWith(t, dir, older, domain.InstallConfig{}))
	require.NoError(t, installWith(t, dir, newer, domain.InstallConfig{}))
	target := filepath.Join(dir, ".strategist", filepath.FromSlash(downgradePath))

	err := installWith(t, dir, older, domain.InstallConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime_newer_than_binary")
	assert.Contains(t, err.Error(), "--allow-downgrade")
	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	assert.Equal(t, downgradePath+" v2\n", string(got), "a refused downgrade changes no file")

	require.Error(t, installWith(t, dir, older, domain.InstallConfig{Force: true}), "--force alone does not allow a downgrade")

	require.NoError(t, installWith(t, dir, older, domain.InstallConfig{AllowDowngrade: true}))
	got, readErr = os.ReadFile(target)
	require.NoError(t, readErr)
	assert.Equal(t, downgradePath+" v1\n", string(got), "an explicit rollback is allowed")
}

func TestInstall_GenuineUpgradeStillAutoUpgrades(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, installWith(t, dir, newRuntimeDefaultsExtractor(nil), domain.InstallConfig{}))
	require.NoError(t, installWith(t, dir, newRuntimeDefaultsExtractor(map[string]string{downgradePath: downgradePath + " v2\n"}), domain.InstallConfig{}))
	require.NoError(t, installWith(t, dir, newRuntimeDefaultsExtractor(map[string]string{downgradePath: downgradePath + " v3\n"}), domain.InstallConfig{}),
		"a default this runtime never held is an upgrade")

	manifest, loaded, err := loadInstallManifest(filepath.Join(dir, ".strategist"))
	require.NoError(t, err)
	require.True(t, loaded)
	entry, ok := manifest.FileByPath(downgradePath)
	require.True(t, ok)
	assert.Equal(t, []string{domain.SHA256Hex([]byte(downgradePath + " v2\n")), domain.SHA256Hex([]byte(downgradePath + " v1\n"))}, entry.History,
		"history lists earlier installed hashes, most recent first")
}

func upgradeWith(t *testing.T, dir, content string, allow bool) error {
	t.Helper()
	svc := upgradeTestService(map[string][]byte{downgradePath: []byte(content)})
	svc.AllowDowngrade = allow
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	return err
}

// `strategist upgrade` applies the same guard as install, before any write.
func TestApplyUpgrade_RefusesADowngradeFromAnOlderBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, upgradeWith(t, dir, "v1", false))
	require.NoError(t, upgradeWith(t, dir, "v2", false))

	err := upgradeWith(t, dir, "v1", false)
	require.ErrorContains(t, err, "runtime_newer_than_binary")
	got, readErr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(downgradePath)))
	require.NoError(t, readErr)
	assert.Equal(t, "v2", string(got))

	require.NoError(t, upgradeWith(t, dir, "v1", true), "--allow-downgrade permits the rollback")
}

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

const customizedConfiguration = "mode: epic\nbase_path: .custom\nslots:\n  discovery: brainstorming\n  refinement: writing-plans\n  execution: sniper\n"

// customizedLock is the bytes writePluginLockFile produces for the fixture.
func customizedLock(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, writePluginLockFile(dir, domain.PluginLockFile{}))
	raw, err := os.ReadFile(filepath.Join(dir, "plugins.lock"))
	require.NoError(t, err)
	return string(raw)
}

// customizedWorkspace installs once and then customizes the three
// configuration files, returning the strategist dir and a service.
func customizedWorkspace(t *testing.T) (string, string, Service) {
	t.Helper()
	dir := t.TempDir()
	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	strategistDir := filepath.Join(dir, ".strategist")
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte(customizedConfiguration), 0o644))
	require.NoError(t, writePluginLockFile(strategistDir, domain.PluginLockFile{}))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, ".config.lock"), []byte("custom-lock\n"), 0o644))
	return dir, strategistDir, svc
}

// This is the reset of 2026-10-11 15:46: a forced silent install rewrote a
// customized active.yaml from the epic template and the binding went back to
// openspec-propose.
func TestForcedSilentInstallOverACustomizedActiveYAMLKeepsTheSlots(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true}))

	got, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	assert.Equal(t, []byte(customizedConfiguration), got, "--force repairs the runtime and must not rewrite the configuration")
	lock, err := os.ReadFile(filepath.Join(strategistDir, "plugins.lock"))
	require.NoError(t, err)
	assert.Equal(t, customizedLock(t), string(lock))
}

func TestResetConfigRewritesTheConfigurationFromTheTemplate(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, ResetConfig: true}))

	got, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	assert.NotEqual(t, customizedConfiguration, string(got))
	assert.Contains(t, string(got), "mode: epic\nbase_path: .analysis\n")
}

func TestResetConfigBacksUpTheConfigurationFilesFirst(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, ResetConfig: true}))

	stamps, err := ListUpgradeBackups(strategistDir)
	require.NoError(t, err)
	require.NotEmpty(t, stamps)
	saved, err := os.ReadFile(filepath.Join(strategistDir, upgradeBackupRelDir, stamps[0], "active.yaml"))
	require.NoError(t, err)
	assert.Equal(t, []byte(customizedConfiguration), saved)
	for _, name := range []string{"plugins.lock", ".config.lock"} {
		_, statErr := os.Stat(filepath.Join(strategistDir, upgradeBackupRelDir, stamps[0], name))
		assert.NoError(t, statErr, name)
	}
}

func TestUpgradeRollbackRestoresTheConfigurationBackup(t *testing.T) {
	t.Parallel()
	dir, strategistDir, svc := customizedWorkspace(t)
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, ResetConfig: true}))
	stamps, err := ListUpgradeBackups(strategistDir)
	require.NoError(t, err)
	require.NotEmpty(t, stamps)

	_, err = RollbackUpgrade(strategistDir, stamps[0])
	require.NoError(t, err)
	restored, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	assert.Equal(t, []byte(customizedConfiguration), restored, "the restored configuration must be byte-identical")
	lock, err := os.ReadFile(filepath.Join(strategistDir, "plugins.lock"))
	require.NoError(t, err)
	assert.Equal(t, customizedLock(t), string(lock))
}

func TestForcedInstallWithoutAnExistingConfigurationStillWritesTheTemplate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := Service{Extractor: minimalExtractor{}, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, Force: true}))
	_, err := os.Stat(filepath.Join(dir, ".strategist", "active.yaml"))
	require.NoError(t, err)
}

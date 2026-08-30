package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetUpgradeFlags(t *testing.T) {
	t.Helper()
	origTarget, origGlobal, origDry, origForce, origRollback :=
		upgradeTarget, upgradeGlobal, upgradeDryRun, upgradeForce, upgradeRollback
	t.Cleanup(func() {
		upgradeTarget, upgradeGlobal, upgradeDryRun, upgradeForce, upgradeRollback =
			origTarget, origGlobal, origDry, origForce, origRollback
	})
	upgradeTarget, upgradeGlobal, upgradeDryRun, upgradeForce, upgradeRollback = "", false, false, false, ""
}

// installedTempDir runs a real silent install into a fresh temp dir and
// returns it, so upgrade tests exercise the actual embedded default tree
// instead of a synthetic fixture.
func installedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	origTarget := installTarget
	t.Cleanup(func() { installTarget = origTarget })
	installTarget = dir

	require.NoError(t, installCmd.RunE(installCmd, nil))
	return dir
}

func TestUpgradeCmd_DryRunOnFreshInstallReportsAllManaged(t *testing.T) {
	resetUpgradeFlags(t)
	dir := installedTempDir(t)

	upgradeTarget = dir
	upgradeDryRun = true

	var out bytes.Buffer
	upgradeCmd.SetOut(&out)
	t.Cleanup(func() { upgradeCmd.SetOut(nil) })

	err := upgradeCmd.RunE(upgradeCmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "managed (no change):")
	assert.Contains(t, out.String(), "dry run — nothing written")

	// Nothing should have moved.
	assert.NoFileExists(t, filepath.Join(dir, ".strategist", ".upgrade-backups"))
}

func TestUpgradeCmd_ForceThenRollbackRoundTrip(t *testing.T) {
	resetUpgradeFlags(t)
	dir := installedTempDir(t)

	target := filepath.Join(dir, ".strategist", "SKILL.md")
	original, err := os.ReadFile(target)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, append(original, []byte("\n# user note\n")...), 0o644))

	upgradeTarget = dir
	upgradeForce = true
	var out bytes.Buffer
	upgradeCmd.SetOut(&out)
	t.Cleanup(func() { upgradeCmd.SetOut(nil) })

	require.NoError(t, upgradeCmd.RunE(upgradeCmd, nil))
	assert.Contains(t, out.String(), "Backed up overwritten files to")

	reverted, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, string(original), string(reverted), "--force must overwrite the customized file")

	upgradeForce = false
	upgradeRollback = "latest"
	out.Reset()
	require.NoError(t, upgradeCmd.RunE(upgradeCmd, nil))

	restored, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(restored), "# user note", "rollback latest must restore the customization")
}

func TestUpgradeCmd_RollbackWithNoBackupsFails(t *testing.T) {
	resetUpgradeFlags(t)
	dir := installedTempDir(t)

	upgradeTarget = dir
	upgradeRollback = "latest"

	err := upgradeCmd.RunE(upgradeCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backups found")
}

func TestUpgradeCmd_UnknownTargetPropagatesError(t *testing.T) {
	resetUpgradeFlags(t)
	clearHomeEnv(t)

	upgradeGlobal = true
	err := upgradeCmd.RunE(upgradeCmd, nil)
	require.Error(t, err)
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRealUpgradeCmd builds the upgrade command with the production wiring, so
// these tests keep proving the real embedded tree and install service work
// end to end; the adapter's own behavior is covered with fakes in
// cmd/strategist/install.
func newRealUpgradeCmd(t *testing.T, flags map[string]string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := installadapter.NewUpgrade(upgradeDependencies())
	for name, value := range flags {
		require.NoError(t, cmd.Flags().Set(name, value), name)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	return cmd, &out
}

// installedTempDir runs a real silent install into a fresh temp dir and
// returns it, so upgrade tests exercise the actual embedded default tree
// instead of a synthetic fixture.
func installedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmd := &cobra.Command{Use: "install"}
	require.NoError(t, installadapter.RunForTest(cmd, installDependencies(), installadapter.TestOptions{Target: dir, NoShim: true}))
	return dir
}

func TestUpgradeCmd_DryRunOnFreshInstallReportsAllManaged(t *testing.T) {
	dir := installedTempDir(t)
	cmd, out := newRealUpgradeCmd(t, map[string]string{"target": dir, "dry-run": "true"})

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, out.String(), "managed (no change):")
	assert.Contains(t, out.String(), "dry run — nothing written")

	// Nothing should have moved.
	assert.NoFileExists(t, filepath.Join(dir, ".strategist", ".upgrade-backups"))
}

func TestUpgradeCmd_ForceThenRollbackRoundTrip(t *testing.T) {
	dir := installedTempDir(t)

	target := filepath.Join(dir, ".strategist", "SKILL.md")
	original, err := os.ReadFile(target)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, append(original, []byte("\n# user note\n")...), 0o644))

	forceCmd, out := newRealUpgradeCmd(t, map[string]string{"target": dir, "force": "true"})
	require.NoError(t, forceCmd.RunE(forceCmd, nil))
	assert.Contains(t, out.String(), "Backed up overwritten files to")

	reverted, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, string(original), string(reverted), "--force must overwrite the customized file")

	rollbackCmd, _ := newRealUpgradeCmd(t, map[string]string{"target": dir, "rollback": "latest"})
	require.NoError(t, rollbackCmd.RunE(rollbackCmd, nil))

	restored, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(restored), "# user note", "rollback latest must restore the customization")
}

func TestUpgradeCmd_RollbackWithNoBackupsFails(t *testing.T) {
	dir := installedTempDir(t)
	cmd, _ := newRealUpgradeCmd(t, map[string]string{"target": dir, "rollback": "latest"})

	err := cmd.RunE(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backups found")
}

func TestUpgradeCmd_UnknownTargetPropagatesError(t *testing.T) {
	clearHomeEnv(t)
	cmd, _ := newRealUpgradeCmd(t, map[string]string{"global": "true"})

	require.Error(t, cmd.RunE(cmd, nil))
}

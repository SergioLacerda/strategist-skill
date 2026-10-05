package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	upgradeTemplateOnce sync.Once
	upgradeTemplateRoot string
	upgradeTemplateErr  error
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

// installedTempDir copies a package-scoped real install into a fresh temp dir,
// so upgrade tests exercise the actual embedded default tree without spawning
// the ranked runtime once per test.
func installedTempDir(t *testing.T) string {
	t.Helper()
	upgradeTemplateOnce.Do(func() {
		upgradeTemplateRoot, upgradeTemplateErr = os.MkdirTemp("", "strategist-upgrade-template-")
		if upgradeTemplateErr != nil {
			return
		}
		target := filepath.Join(upgradeTemplateRoot, "install")
		cmd := &cobra.Command{Use: "install"}
		upgradeTemplateErr = installadapter.RunForTest(cmd, installDependencies(), installadapter.TestOptions{Target: target, NoShim: true})
		if upgradeTemplateErr == nil {
			upgradeTemplateRoot = target
		}
	})
	require.NoError(t, upgradeTemplateErr)

	dir := t.TempDir()
	require.NoError(t, copyUpgradeTree(upgradeTemplateRoot, dir))
	activePath := filepath.Join(dir, ".strategist", "active.yaml")
	lockPath := filepath.Join(dir, ".strategist", ".config.lock")
	require.NoError(t, integrity.WriteLock(activePath, lockPath))
	lock, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(lock), dir, "the copied config lock must point at the copy")
	assert.NotContains(t, string(lock), upgradeTemplateRoot, "the copied config lock must not retain the template path")
	return dir
}

// cleanupUpgradeTemplate is called by TestMain after m.Run. The template is
// deliberately outside t.TempDir because it must outlive the first test that
// initializes the sync.Once and be shared by all three upgrade tests.
func cleanupUpgradeTemplate() {
	if upgradeTemplateRoot == "" {
		return
	}
	root := upgradeTemplateRoot
	if filepath.Base(root) == "install" {
		root = filepath.Dir(root)
	}
	_ = os.RemoveAll(root)
}

func copyUpgradeTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	return filepath.WalkDir(src, upgradeTreeCopier{src: src, dst: dst}.walk)
}

// upgradeTreeCopier is the walk callback of copyUpgradeTree as a method, so its
// branching is not nested inside a closure of the function that starts the walk.
type upgradeTreeCopier struct{ src, dst string }

func (c upgradeTreeCopier) walk(path string, entry os.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if path == c.src {
		return nil
	}
	rel, err := filepath.Rel(c.src, path)
	if err != nil {
		return err
	}
	return copyUpgradeEntry(path, filepath.Join(c.dst, rel), entry)
}

func copyUpgradeEntry(src, dst string, entry os.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if entry.Type()&os.ModeSymlink != 0 {
		link, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(link, dst)
	}
	if entry.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		return os.Chmod(dst, info.Mode().Perm())
	}
	return copyUpgradeFile(src, dst, info.Mode().Perm())
}

func copyUpgradeFile(src, dst string, mode os.FileMode) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func TestUpgradeCmd_DryRunOnFreshInstallReportsAllManaged(t *testing.T) {
	t.Parallel()
	dir := installedTempDir(t)
	cmd, out := newRealUpgradeCmd(t, map[string]string{"target": dir, "dry-run": "true"})

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, out.String(), "managed (no change):")
	assert.Contains(t, out.String(), "dry run — nothing written")

	// Nothing should have moved.
	assert.NoFileExists(t, filepath.Join(dir, ".strategist", ".upgrade-backups"))
}

func TestUpgradeCmd_ForceThenRollbackRoundTrip(t *testing.T) {
	t.Parallel()
	dir := installedTempDir(t)
	templateLock, err := os.ReadFile(filepath.Join(upgradeTemplateRoot, ".strategist", ".config.lock"))
	require.NoError(t, err)

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
	finalTemplateLock, err := os.ReadFile(filepath.Join(upgradeTemplateRoot, ".strategist", ".config.lock"))
	require.NoError(t, err)
	assert.Equal(t, string(templateLock), string(finalTemplateLock), "force and rollback must not mutate the template")
}

func TestUpgradeCmd_RollbackWithNoBackupsFails(t *testing.T) {
	t.Parallel()
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

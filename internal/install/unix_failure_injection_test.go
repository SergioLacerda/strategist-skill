//go:build !windows

package install

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// Permission- and FIFO-based failure injection is only meaningful on unix
// hosts, and permission bits do not bind root.
func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("permission bits do not restrict root")
	}
}

func TestCopySkillPackageRejectsNonRegularEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o600))
	require.ErrorContains(t, copySkillPackage(dir, t.TempDir()), "unsupported package entry")
}

func TestCommitStagedDirectoryPermissionFailures(t *testing.T) {
	skipIfRoot(t)
	locked := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(locked, ".staging"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(locked, "inst", "child"), 0o755))
	require.NoError(t, os.Chmod(locked, 0o500))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) }) //nolint:errcheck // test cleanup
	require.ErrorContains(t, commitStagedDirectory(filepath.Join(locked, "inst"), map[string][]byte{"a.yaml": []byte("x")}), "replace installed provider")

	noStaging := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(noStaging, ".staging"), 0o500))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(noStaging, ".staging"), 0o755) }) //nolint:errcheck // test cleanup
	require.ErrorContains(t, commitStagedDirectory(filepath.Join(noStaging, "inst"), nil), "create provider staging directory")
}

func TestNormalizedSkillDigestUnreadableFile(t *testing.T) {
	skipIfRoot(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0o000))
	_, err := normalizedSkillDigest(dir, nil)
	require.ErrorContains(t, err, "read")
}

//go:build windows

package filelock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsLockHelpersCoverNilAndCleanupFailures(t *testing.T) {
	require.ErrorContains(t, lockFile(nil), "nil lock file")
	require.ErrorContains(t, unlockFile(nil), "nil lock file")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child"), []byte("busy"), 0o600))
	directory, err := os.Open(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = directory.Close() })
	require.ErrorContains(t, removeLock(directory), "remove lock file")
}

func TestWindowsOpenLockTimeoutCanBeConfiguredForTests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.lock")
	blocker, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = blocker.Close(); _ = os.Remove(path) })

	previous := lockPollTimeout
	lockPollTimeout = 0
	t.Cleanup(func() { lockPollTimeout = previous })

	_, err = openLock(path)
	require.ErrorContains(t, err, "timed out")
}

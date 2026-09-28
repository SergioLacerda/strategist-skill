//go:build !windows

package filelock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenLockWrapsOpenFailure(t *testing.T) {
	_, err := openLock(filepath.Join(t.TempDir(), "missing", "lock"))
	require.ErrorContains(t, err, "filelock: open lock file")
}

func TestLockFileWrapsFlockFailure(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lock")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// flock on an already-closed file descriptor fails (EBADF).
	require.ErrorContains(t, lockFile(f), "filelock: lock")
}

func TestUnlockFileWrapsFlockFailure(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lock")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.ErrorContains(t, unlockFile(f), "filelock: unlock")
}

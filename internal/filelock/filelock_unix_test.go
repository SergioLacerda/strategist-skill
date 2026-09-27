//go:build !windows

package filelock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenLockWrapsOpenFailure(t *testing.T) {
	_, err := openLock(filepath.Join(t.TempDir(), "missing", "lock"))
	require.ErrorContains(t, err, "filelock: open lock file")
}

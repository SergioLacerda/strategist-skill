package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomicReplacesContentAndLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	require.NoError(t, WriteFileAtomic(path, []byte("new"), 0o640))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new", string(got))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestWriteFileAtomicFailsWithoutParentDirectory(t *testing.T) {
	require.Error(t, WriteFileAtomic(filepath.Join(t.TempDir(), "missing", "f"), []byte("x"), 0o600))
}

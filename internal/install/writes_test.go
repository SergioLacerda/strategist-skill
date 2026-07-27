package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFile_MkdirError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A file where a directory component is expected makes MkdirAll fail.
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := atomicWriteFile(filepath.Join(blocker, "sub", "file.txt"), []byte("data"), 0o644)
	require.Error(t, err)
	assert.ErrorContains(t, err, "mkdir parent")
}

package embed

// Whitebox tests for extractFS — covers all error paths using an
// in-memory fs.FS constructed with testing/fstest.

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFS_Basic(t *testing.T) {
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
		"root/sub/a.md":  {Data: []byte("# A")},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir))
	assert.FileExists(t, filepath.Join(dir, "file.yaml"))
	assert.FileExists(t, filepath.Join(dir, "sub", "a.md"))
}

func TestExtractFS_MkdirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/sub/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := extractFS(memFS, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed:")
}

func TestExtractFS_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	// Pre-create file.yaml as a directory so WriteFile fails with EISDIR
	require.NoError(t, os.Mkdir(filepath.Join(dir, "file.yaml"), 0o755))

	err := extractFS(memFS, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: write")
}

func TestExtractFS_EmptyRoot(t *testing.T) {
	memFS := fstest.MapFS{
		"root": {Mode: os.ModeDir},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir))
}

// errFS is an fs.FS that wraps a MapFS but fails ReadDir for a specific path,
// causing fs.WalkDir to deliver an error to the walk callback.
type errFS struct {
	fstest.MapFS
	failPath string
}

func (e errFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == e.failPath {
		return nil, os.ErrPermission
	}
	return e.MapFS.ReadDir(name)
}

func TestExtractFS_WalkCallbackError(t *testing.T) {
	// errFS makes WalkDir deliver an error for "root/broken" to the callback.
	mem := errFS{
		MapFS: fstest.MapFS{
			"root/broken": {Mode: os.ModeDir},
		},
		failPath: "root/broken",
	}
	dir := t.TempDir()
	err := extractFS(mem, "root", dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: walk")
}

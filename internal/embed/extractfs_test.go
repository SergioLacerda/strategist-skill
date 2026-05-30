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
	t.Parallel()
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
		"root/sub/a.md":  {Data: []byte("# A")},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir, false))
	assert.FileExists(t, filepath.Join(dir, "file.yaml"))
	assert.FileExists(t, filepath.Join(dir, "sub", "a.md"))
}

func TestExtractFS_MkdirError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/sub/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := extractFS(memFS, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed:")
}

func TestExtractFS_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	memFS := fstest.MapFS{
		"root/file.yaml": {Data: []byte("x: 1\n")},
	}
	dir := t.TempDir()
	// Pre-create file.yaml as a directory so WriteFile fails with EISDIR
	require.NoError(t, os.Mkdir(filepath.Join(dir, "file.yaml"), 0o755))

	err := extractFS(memFS, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: write")
}

func TestExtractFS_EmptyRoot(t *testing.T) {
	t.Parallel()
	memFS := fstest.MapFS{
		"root": {Mode: os.ModeDir},
	}
	dir := t.TempDir()
	require.NoError(t, extractFS(memFS, "root", dir, false))
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
	t.Parallel()
	// errFS makes WalkDir deliver an error for "root/broken" to the callback.
	mem := errFS{
		MapFS: fstest.MapFS{
			"root/broken": {Mode: os.ModeDir},
		},
		failPath: "root/broken",
	}
	dir := t.TempDir()
	err := extractFS(mem, "root", dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed: walk")
}

// --- Merge mode tests ---

// TestMergeMode_PreservesUserModified verifies that re-extracting with force=false
// does not overwrite a file the user has modified.
func TestMergeMode_PreservesUserModified(t *testing.T) {
	t.Parallel()
	embedded := []byte("default: value\n")
	userContent := []byte("user: customized\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: embedded},
	}
	dir := t.TempDir()

	// First install: file doesn't exist → written from embedded.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	configPath := filepath.Join(dir, "config.yaml")
	got, _ := os.ReadFile(configPath)
	assert.Equal(t, embedded, got, "first install should write embedded content")

	// User customizes the file.
	require.NoError(t, os.WriteFile(configPath, userContent, 0o644))

	// Re-install (merge mode): user content must be preserved.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	got, _ = os.ReadFile(configPath)
	assert.Equal(t, userContent, got, "merge mode must preserve user-modified file")
}

// TestMergeMode_Idempotent verifies that re-extracting with force=false and the
// same embedded content is idempotent — the file is overwritten with identical bytes.
func TestMergeMode_Idempotent(t *testing.T) {
	t.Parallel()
	content := []byte("version: 1\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: content},
	}
	dir := t.TempDir()

	// First install.
	require.NoError(t, extractFS(memFS, "root", dir, false))

	// Re-install with same embedded content: should succeed without error.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	assert.Equal(t, content, got, "idempotent re-install should keep the same content")
}

// TestMergeMode_NewEmbeddedVersionPreservesOnDisk documents a known limitation:
// when the embedded version changes (v1→v2) but the on-disk file is still v1
// (user never customized it), merge mode cannot distinguish this from a user
// modification and skips the update. Use --force to get new embedded defaults.
func TestMergeMode_NewEmbeddedVersionPreservesOnDisk(t *testing.T) {
	t.Parallel()
	v1 := []byte("version: 1\n")
	v2 := []byte("version: 2\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: v1},
	}
	dir := t.TempDir()

	// First install writes v1.
	require.NoError(t, extractFS(memFS, "root", dir, false))

	// Embedded bumped to v2 (simulates a new release).
	memFS["root/config.yaml"] = &fstest.MapFile{Data: v2}

	// Merge mode cannot distinguish "user kept v1" from "embedded upgraded" —
	// on-disk v1 differs from new embedded v2, so the file is preserved.
	require.NoError(t, extractFS(memFS, "root", dir, false))
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	assert.Equal(t, v1, got, "limitation: merge mode preserves on-disk v1 even when embedded moves to v2")
}

// TestForceMode_OverwritesUserModified verifies that force=true always overwrites.
func TestForceMode_OverwritesUserModified(t *testing.T) {
	t.Parallel()
	embedded := []byte("default: value\n")
	userContent := []byte("user: customized\n")
	memFS := fstest.MapFS{
		"root/config.yaml": {Data: embedded},
	}
	dir := t.TempDir()

	// Write user content directly (simulating a customized install).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), userContent, 0o644))

	// Force extract: must overwrite user content.
	require.NoError(t, extractFS(memFS, "root", dir, true))
	got, _ := os.ReadFile(filepath.Join(dir, "config.yaml"))
	assert.Equal(t, embedded, got, "force mode must overwrite user-modified files")
}

// TestUserModified_NonExistent verifies userModified returns false for a file that
// doesn't exist on disk (not yet installed — not a user modification).
func TestUserModified_NonExistent(t *testing.T) {
	t.Parallel()
	result := userModified("/tmp/definitely-does-not-exist-xyz123.yaml", []byte("x: 1\n"))
	assert.False(t, result, "non-existent file is not user-modified")
}

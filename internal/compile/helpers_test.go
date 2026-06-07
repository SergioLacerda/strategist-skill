package compile

// Whitebox tests for unexported helpers — kept minimal, covering error paths
// that cannot be triggered through the public API.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMtime_MissingFile(t *testing.T) {
	t.Parallel()
	result := mtime(filepath.Join(t.TempDir(), "nonexistent.txt"))
	assert.Equal(t, int64(0), result, "mtime of missing file must return 0")
}

func TestSHA256Artifact_MissingFile(t *testing.T) {
	t.Parallel()
	result := sha256Artifact(filepath.Join(t.TempDir(), "nonexistent.gz"))
	assert.Equal(t, "unavailable", result)
}

func TestWriteGzJSON_NonSerializable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	out := filepath.Join(dir, "out.gz")
	// Channels cannot be JSON-encoded — triggers the json.Encode error path
	err := writeGzJSON(out, make(chan int))
	require.Error(t, err)
}

func TestCompileYAMLDir_UnreadableDir(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	subdir := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(subdir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.yaml"), []byte("x: 1\n"), 0o644))
	require.NoError(t, os.Chmod(subdir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })

	sources := map[string]int64{}
	_, err := compileYAMLDir(subdir, sources)
	require.Error(t, err)
}

func TestWriteGzJSON_MkdirFails(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	// Path under read-only dir — MkdirAll cannot create subdir
	err := writeGzJSON(filepath.Join(dir, "subdir", "out.gz"), map[string]string{"x": "y"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "mkdir")
}

func TestCompileYAMLDirTyped_UnreadableDir(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	subdir := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(subdir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.yaml"), []byte("x: 1\n"), 0o644))
	require.NoError(t, os.Chmod(subdir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })

	_, err := compileYAMLDirTyped[map[string]any](subdir, nil)
	require.Error(t, err)
}

func TestCompileYAMLDirTyped_WithSources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "item.yaml"), []byte("key: value\n"), 0o644))

	sources := map[string]int64{}
	result, err := compileYAMLDirTyped[map[string]any](dir, sources)
	require.NoError(t, err)
	assert.Contains(t, result, "item")
	assert.NotEmpty(t, sources, "sources map must be populated when non-nil")
}

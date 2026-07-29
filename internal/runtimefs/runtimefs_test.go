package runtimefs_test

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	assert.False(t, runtimefs.Exists(filepath.Join(dir, "missing")))
	assert.True(t, runtimefs.Exists(dir))
}

func TestReadSHA256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	data := []byte("content")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	got, exists, err := runtimefs.ReadSHA256(path)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(data)), got)

	got, exists, err = runtimefs.ReadSHA256(filepath.Join(dir, "missing.txt"))
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Empty(t, got)
}

func TestWriteFileCreatesParent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "file.txt")
	require.NoError(t, runtimefs.WriteFile(path, []byte("content"), 0o644))
	assert.FileExists(t, path)
}

func TestWriteGzJSONCreatesParentAndWritesArchive(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "artifact.gz")
	require.NoError(t, runtimefs.WriteGzJSON(path, map[string]string{"key": "value"}))

	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, gz.Close()) })

	var got map[string]string
	require.NoError(t, json.NewDecoder(gz).Decode(&got))
	assert.Equal(t, map[string]string{"key": "value"}, got)
}

func TestWriteGzJSONCleansTempFileOnEncodeError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifact.gz")
	err := runtimefs.WriteGzJSON(path, make(chan int))
	require.Error(t, err)
	assert.NoFileExists(t, path)
	assert.NoFileExists(t, path+".tmp")
}

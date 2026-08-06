package runtimefs_test

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

func TestReadSHA256_RealErrorOnDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, exists, err := runtimefs.ReadSHA256(dir)
	require.Error(t, err)
	assert.False(t, exists)
}

func TestWriteFile_MkdirAllFailsWhenParentIsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := runtimefs.WriteFile(filepath.Join(blocker, "nested", "file.txt"), []byte("content"), 0o644)
	require.Error(t, err)
}

func TestWriteFile_WriteFailsWhenPathIsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "is-a-dir")
	require.NoError(t, os.Mkdir(target, 0o755))

	err := runtimefs.WriteFile(target, []byte("content"), 0o644)
	require.Error(t, err)
}

func TestWriteGzJSON_MkdirAllFailsWhenParentIsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := runtimefs.WriteGzJSON(filepath.Join(blocker, "nested", "artifact.gz"), map[string]string{"k": "v"})
	require.Error(t, err)
}

func TestWriteGzJSON_CreateFailsWhenTempPathIsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "artifact.gz")
	// The tmp path (target + ".tmp") is pre-occupied by a directory, so
	// os.Create(tmp) fails — target itself stays a normal, nonexistent path.
	require.NoError(t, os.Mkdir(target+".tmp", 0o755))

	err := runtimefs.WriteGzJSON(target, map[string]string{"k": "v"})
	require.Error(t, err)
}

// TestWriteGzJSON_GzipCloseWriteErrorPropagates triggers gz.Close's flush
// write to fail deterministically via RLIMIT_FSIZE — the file grows past
// the process file-size limit partway through the gzip footer flush.
// Deliberately not t.Parallel(): it mutates a process-wide rlimit, restored
// via t.Cleanup before returning; Go's testing framework runs all
// non-parallel top-level tests to completion before any t.Parallel() test
// in this file begins its body, so this cannot race with its siblings.
func TestWriteGzJSON_GzipCloseWriteErrorPropagates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("RLIMIT_FSIZE behavior is Linux-specific")
	}
	var original syscall.Rlimit
	require.NoError(t, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &original))
	t.Cleanup(func() {
		require.NoError(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &original))
	})
	require.NoError(t, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 100, Max: original.Max}))

	path := filepath.Join(t.TempDir(), "artifact.gz")
	err := runtimefs.WriteGzJSON(path, map[string]string{"k": strings.Repeat("x", 100_000)})
	require.Error(t, err)
	assert.NoFileExists(t, path)
	assert.NoFileExists(t, path+".tmp")
}

func TestWriteGzJSON_RenameFailsWhenPathIsDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// path is an existing directory: the tmp file (path+".tmp") is created
	// and encoded successfully, but the final os.Rename onto an existing
	// directory fails — same technique already proven for
	// internal/telemetry's SaveCheckpoint and internal/integrity's
	// WriteLock in this same coverage-improvement pass.
	target := filepath.Join(dir, "is-a-dir")
	require.NoError(t, os.Mkdir(target, 0o755))

	err := runtimefs.WriteGzJSON(target, map[string]string{"k": "v"})
	require.Error(t, err)
	assert.NoFileExists(t, target+".tmp")
}

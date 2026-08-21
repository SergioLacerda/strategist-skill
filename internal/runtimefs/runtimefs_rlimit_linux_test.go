//go:build linux

package runtimefs_test

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteGzJSON_GzipCloseWriteErrorPropagates triggers gz.Close's flush
// write to fail deterministically via RLIMIT_FSIZE — the file grows past
// the process file-size limit partway through the gzip footer flush.
// Deliberately not t.Parallel(): it mutates a process-wide rlimit, restored
// via t.Cleanup before returning; Go's testing framework runs all
// non-parallel top-level tests to completion before any t.Parallel() test
// in this file begins its body, so this cannot race with its siblings.
//
// syscall.Rlimit/Getrlimit/Setrlimit/RLIMIT_FSIZE are Linux-only symbols —
// this file carries a //go:build linux tag (not just a runtime.GOOS check)
// so the package still compiles on Windows/macOS.
func TestWriteGzJSON_GzipCloseWriteErrorPropagates(t *testing.T) {
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

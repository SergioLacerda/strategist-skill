package filelock

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTestFailure = errors.New("filelock: injected test failure")

// TestWithLock_MutualExclusion runs many goroutines that each increment a
// shared, unsynchronized counter inside WithLock. Without mutual exclusion
// this is a data race that usually still "works" on a single field; the real
// assertion is the interleaved-read/write pattern below, which only survives
// under true serialization.
func TestWithLock_MutualExclusion(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state")

	const workers = 16
	const incrementsPerWorker = 50
	var counter int64
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		startLockWorker(&wg, path, incrementsPerWorker, &counter, errs)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int64(workers*incrementsPerWorker), atomic.LoadInt64(&counter))
}

func startLockWorker(wg *sync.WaitGroup, path string, increments int, counter *int64, errs chan<- error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < increments; j++ {
			err := WithLock(path, func() error {
				current := atomic.LoadInt64(counter)
				atomic.StoreInt64(counter, current+1)
				return nil
			})
			if err != nil {
				errs <- err
				return
			}
		}
	}()
}

func TestWithLock_ReleasesOnError(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state")

	err := WithLock(path, func() error { return errTestFailure })
	require.ErrorIs(t, err, errTestFailure)

	// A second acquisition must not deadlock if the first released properly.
	acquired := false
	err = WithLock(path, func() error { acquired = true; return nil })
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestWithLock_CreatesParentDirectory(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "dir", "state")

	err := WithLock(path, func() error { return nil })
	require.NoError(t, err)
}

func TestWithLock_WrapsMkdirAllFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))

	// The lock directory's parent is a regular file, so MkdirAll cannot
	// create "blocker/sub" beneath it.
	path := filepath.Join(blocker, "sub", "state")
	err := WithLock(path, func() error { return nil })
	require.ErrorContains(t, err, "filelock: create lock directory")
}

func TestWithLock_WrapsOpenLockFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "state")
	// Pre-create the lock path itself as a directory: opening it O_RDWR fails.
	require.NoError(t, os.Mkdir(path+".lock", 0o750))

	err := WithLock(path, func() error { return nil })
	require.ErrorContains(t, err, "filelock: open lock")
}

func TestCloseLock_ReportsUnlockAndCloseFailures(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "lock")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// The file descriptor is already closed: unlocking and closing it again
	// both fail, and closeLock reports both rather than only the first.
	err = closeLock(f, true)
	require.Error(t, err)
}

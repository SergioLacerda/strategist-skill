package filelock

import (
	"errors"
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

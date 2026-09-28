//go:build windows

package filelock

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	// lockPollInterval and lockPollTimeout bound the retry loop openLock
	// uses in place of a blocking primitive (see openLock's doc comment).
	lockPollInterval = 5 * time.Millisecond
	lockPollTimeout  = 30 * time.Second
)

// openLock retries exclusive lock-file creation until it succeeds or
// lockPollTimeout elapses. Unix's flock(LOCK_EX) blocks until the lock is
// available, which is what gives WithLock its mutual-exclusion contract
// (concurrent callers serialize; none fail). Windows has no equivalent
// blocking primitive available through the standard library, so this poll
// loop is this platform's stand-in: without it, a caller that finds the
// lock file already present would fail immediately instead of waiting its
// turn, breaking that same contract on Windows only.
func openLock(path string) (*os.File, error) {
	deadline := time.Now().Add(lockPollTimeout)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("filelock: open lock file: %w", err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("filelock: timed out after %s waiting for exclusive lock: %w", lockPollTimeout, err)
		}
		time.Sleep(lockPollInterval)
	}
}

func removeLock(file *os.File) error {
	if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("filelock: remove lock file: %w", err)
	}
	return nil
}

// Windows uses an exclusive lock-file creation primitive (openLock above);
// by the time a caller reaches lockFile, the exclusive create has already
// succeeded, so there is nothing further to acquire here.
func lockFile(file *os.File) error {
	if file == nil {
		return fmt.Errorf("filelock: lock: nil lock file")
	}
	return nil
}

func unlockFile(file *os.File) error {
	if file == nil {
		return fmt.Errorf("filelock: unlock: nil lock file")
	}
	return nil
}

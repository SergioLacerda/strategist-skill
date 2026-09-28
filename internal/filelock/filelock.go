// Package filelock provides a cross-platform exclusive file lock used to
// guard a read-modify-write critical section against concurrent CLI
// invocations on the same machine. It is generalized from
// internal/leveling's own ledger lock (ADR-0057 § D2) so mission-state
// persistence and any future caller share one implementation instead of
// duplicating the Unix/Windows split.
//
// This lock serializes local processes via flock (Unix) or exclusive
// lock-file creation (Windows). It does not protect against writers on
// different machines sharing a directory over a network filesystem — see
// ADR-0057 § Consequences for that limitation, accepted rather than
// mitigated.
package filelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WithLock creates path's parent directory if needed, acquires an exclusive
// lock on "<path>.lock", runs fn, and releases the lock — even if fn panics
// or returns an error. The lock is process-exclusive: a second caller for the
// same path blocks until the first releases it.
func WithLock(path string, fn func() error) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("filelock: create lock directory: %w", err)
	}
	lock, err := openLock(path + ".lock")
	if err != nil {
		return fmt.Errorf("filelock: open lock: %w", err)
	}
	if lockErr := lockFile(lock); lockErr != nil {
		return errors.Join(lockErr, closeLock(lock, false))
	}
	defer func() { err = errors.Join(err, closeLock(lock, true)) }()
	return fn()
}

func closeLock(lock *os.File, unlock bool) error {
	var cleanup error
	if unlock {
		cleanup = unlockFile(lock)
	}
	if err := lock.Close(); err != nil {
		cleanup = errors.Join(cleanup, fmt.Errorf("filelock: close lock: %w", err))
	}
	cleanup = errors.Join(cleanup, removeLock(lock))
	return cleanup
}

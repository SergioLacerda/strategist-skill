package integrity

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// lockTempFile is the subset of *os.File that writeLockFileAtomically needs.
// createLockTempFile and chmodLockTempFile below exist only so tests can
// substitute a fault-injecting fake for the Write/Close/Chmod calls on a
// temp file the caller otherwise never gets a handle to — see
// warning_internal_test.go.
type lockTempFile interface {
	io.Writer
	Close() error
	Name() string
}

var createLockTempFile = func(dir, pattern string) (lockTempFile, error) {
	return os.CreateTemp(dir, pattern)
}

var chmodLockTempFile = os.Chmod

// writeLockFileAtomically writes data to a sibling temp file in lockPath's
// directory, then renames it into place, so a process interruption never
// leaves a partially written lock file. It does not create missing parent
// directories: the lock's directory (e.g. .strategist/) is expected to exist.
func writeLockFileAtomically(lockPath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(lockPath)
	tmp, err := createLockTempFile(dir, ".config.lock.tmp-*")
	if err != nil {
		return fmt.Errorf("create temp lock file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup; no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("write temp lock file: %w", errors.Join(err, closeErr))
		}
		return fmt.Errorf("write temp lock file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp lock file: %w", err)
	}
	if err := chmodLockTempFile(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp lock file: %w", err)
	}
	if err := os.Rename(tmpPath, lockPath); err != nil {
		return fmt.Errorf("rename lock into place: %w", err)
	}
	return nil
}

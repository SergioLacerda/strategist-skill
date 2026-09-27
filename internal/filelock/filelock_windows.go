//go:build windows

package filelock

import (
	"fmt"
	"os"
)

func openLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
}

func removeLock(file *os.File) error {
	if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("filelock: remove lock file: %w", err)
	}
	return nil
}

// Windows uses an exclusive lock-file creation primitive. The open handle is
// retained until the operation completes, so competing processes fail closed
// instead of silently entering an unlocked critical section.
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

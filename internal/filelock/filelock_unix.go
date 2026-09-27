//go:build !windows

package filelock

import (
	"fmt"
	"os"
	"syscall"
)

func openLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
	if err != nil {
		return nil, fmt.Errorf("filelock: open lock file: %w", err)
	}
	return file, nil
}

func removeLock(_ *os.File) error { return nil }

func lockFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("filelock: lock: %w", err)
	}
	return nil
}

func unlockFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("filelock: unlock: %w", err)
	}
	return nil
}

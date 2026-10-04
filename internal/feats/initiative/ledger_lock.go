package initiative

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func withLedgerLock(path string, fn func() error) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("initiative: create ledger directory: %w", err)
	}
	lock, err := openLedgerLock(path + ".lock")
	if err != nil {
		return fmt.Errorf("initiative: open ledger lock: %w", err)
	}
	if lockErr := lockLedgerFile(lock); lockErr != nil {
		return errors.Join(lockErr, closeLedgerFile(lock, false))
	}
	defer func() { err = errors.Join(err, closeLedgerFile(lock, true)) }()
	return fn()
}

func closeLedgerFile(lock *os.File, unlock bool) error {
	var cleanup error
	if unlock {
		cleanup = unlockLedgerFile(lock)
	}
	if err := lock.Close(); err != nil {
		cleanup = errors.Join(cleanup, fmt.Errorf("initiative: close ledger lock: %w", err))
	}
	cleanup = errors.Join(cleanup, removeLedgerLock(lock))
	return cleanup
}

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
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
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
	return cleanup
}

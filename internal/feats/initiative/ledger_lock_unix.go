//go:build !windows

package initiative

import (
	"fmt"
	"os"
	"syscall"
)

func openLedgerLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
	if err != nil {
		return nil, fmt.Errorf("initiative: open ledger lock: %w", err)
	}
	return file, nil
}

func removeLedgerLock(_ *os.File) error { return nil }

func lockLedgerFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("initiative: lock ledger: %w", err)
	}
	return nil
}

func unlockLedgerFile(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("initiative: unlock ledger: %w", err)
	}
	return nil
}

//go:build windows

package initiative

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openLedgerLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
	if err != nil {
		return nil, fmt.Errorf("initiative: create ledger lock file: %w", err)
	}
	return file, nil
}

func removeLedgerLock(_ *os.File) error {
	// A competing Windows opener can keep the lock file handle alive after the
	// current owner unlocks it. Removing the pathname then fails with sharing
	// violation; the stable lock file is harmless and is reused on the next run.
	return nil
}

func lockLedgerFile(file *os.File) error {
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped); err != nil {
		return fmt.Errorf("initiative: lock ledger: %w", err)
	}
	return nil
}

func unlockLedgerFile(file *os.File) error {
	overlapped := new(windows.Overlapped)
	if err := windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, overlapped); err != nil {
		return fmt.Errorf("initiative: unlock ledger: %w", err)
	}
	return nil
}

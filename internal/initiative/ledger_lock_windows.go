//go:build windows

package initiative

import (
	"fmt"
	"os"
)

func openLedgerLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
}

func removeLedgerLock(file *os.File) error {
	if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("initiative: remove ledger lock: %w", err)
	}
	return nil
}

// Windows uses exclusive lock-file creation so a competing process fails
// closed rather than entering an unlocked critical section.
func lockLedgerFile(_ *os.File) error   { return nil }
func unlockLedgerFile(_ *os.File) error { return nil }

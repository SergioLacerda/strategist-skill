//go:build windows

package leveling

import (
	"fmt"
	"os"
)

func openLedgerLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600) //nolint:gosec // runtime memory path
}

func removeLedgerLock(file *os.File) error {
	if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("leveling: remove ledger lock: %w", err)
	}
	return nil
}

// Windows uses an exclusive lock-file creation primitive. The open handle is
// retained until the operation completes, so competing processes fail closed
// instead of silently entering an unlocked critical section.
func lockLedgerFile(file *os.File) error {
	if file == nil {
		return fmt.Errorf("leveling: lock ledger: nil lock file")
	}
	return nil
}

func unlockLedgerFile(file *os.File) error {
	if file == nil {
		return fmt.Errorf("leveling: unlock ledger: nil lock file")
	}
	return nil
}

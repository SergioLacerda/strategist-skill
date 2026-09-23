//go:build !windows

package initiative

import (
	"fmt"
	"os"
	"syscall"
)

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

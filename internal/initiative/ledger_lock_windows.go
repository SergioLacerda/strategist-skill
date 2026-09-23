//go:build windows

package initiative

import "os"

// Windows currently has no cross-process lock primitive in the standard
// runtime package. The lock file remains part of the ledger contract so a
// platform-specific implementation can be added without changing callers.
func lockLedgerFile(_ *os.File) error   { return nil }
func unlockLedgerFile(_ *os.File) error { return nil }

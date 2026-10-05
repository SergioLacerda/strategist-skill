package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
)

// withLockedJSONL opens path for read and append (creating the file and its
// directory), holds an exclusive lock while fn runs and always releases it.
// O_APPEND keeps every write at the end of the file on all platforms, so a
// platform whose lock helper is a no-op still never overwrites a line.
// what names the ledger in error messages.
func withLockedJSONL(path, what string, fn func(f *os.File) error) (err error) {
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s dir: %w", what, err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644) //nolint:gosec // G304: ledger path is owned by the runtime memory domain
	if err != nil {
		return fmt.Errorf("open %s: %w", what, err)
	}
	defer closeFileWithContext(f, &err, "close "+what)
	if err = lockFileExclusive(f); err != nil {
		return fmt.Errorf("lock %s: %w", what, err)
	}
	defer func() {
		if unlockErr := unlockFile(f); unlockErr != nil && err == nil {
			err = fmt.Errorf("unlock %s: %w", what, unlockErr)
		}
	}()
	return fn(f)
}

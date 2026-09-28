package leveling

import "github.com/SergioLacerda/strategist-skill/internal/filelock"

// withLedgerLock guards a ledger read-modify-write critical section against
// concurrent CLI invocations. This package's own Unix/Windows flock
// implementation was generalized into internal/filelock (ADR-0057 § D2) so
// mission-state persistence could reuse it instead of duplicating the
// platform split; this is now a thin delegation to keep every existing
// caller in this package unchanged.
func withLedgerLock(path string, fn func() error) error {
	return filelock.WithLock(path, fn) //nolint:wrapcheck // the result is fn()'s own already-meaningful error on the common path; filelock's own lock-acquisition errors are already prefixed "filelock: " internally
}

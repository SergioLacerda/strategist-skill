//go:build !windows

package leveling

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenLedgerLockWrapsOpenFailure(t *testing.T) {
	_, err := openLedgerLock(filepath.Join(t.TempDir(), "missing", "ledger.lock"))
	require.ErrorContains(t, err, "leveling: open ledger lock")
}

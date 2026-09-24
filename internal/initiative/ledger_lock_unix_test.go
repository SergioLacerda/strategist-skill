//go:build !windows

package initiative

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenLedgerLockWrapsOpenFailure(t *testing.T) {
	_, err := openLedgerLock(filepath.Join(t.TempDir(), "missing", "ledger.lock"))
	require.ErrorContains(t, err, "initiative: open ledger lock")
}

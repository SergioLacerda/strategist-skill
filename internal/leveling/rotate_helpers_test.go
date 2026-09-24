package leveling

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerReadErrorClassifiesScannerFailures(t *testing.T) {
	require.NoError(t, ledgerReadError(nil))
	require.ErrorContains(t, ledgerReadError(errors.New("bufio.Scanner: token too long")), "leveling_ledger_record_oversized")
	require.ErrorContains(t, ledgerReadError(errors.New("boom")), "leveling: read role level ledger")
}

func TestKeptLinesWritersReportFailuresOnAClosedFile(t *testing.T) {
	closed := func() *os.File {
		f, err := os.CreateTemp(t.TempDir(), "rotation-*.tmp")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		return f
	}
	require.ErrorContains(t, flushKeptLines(closed(), []string{"a"}, []bool{true}), "write rotation file")
	require.ErrorContains(t, flushKeptLines(closed(), nil, nil), "write rotation file")
	require.ErrorContains(t, writeKeptLines(closed(), []string{"a"}, []bool{true}), "write rotation file")
}

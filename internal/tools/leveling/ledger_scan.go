package leveling

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// scanLedger calls visit for every well-formed line of the ledger, in order. A
// missing ledger is a no-op; malformed lines are skipped so a damaged history
// never blocks a report, a rotation or a mission.
func scanLedger(path string, visit func(line string, record Record)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("leveling: create ledger directory: %w", err)
	}
	return withLedgerLock(path, func() error { return scanLedgerUnlocked(path, visit) })
}

func scanLedgerUnlocked(path string, visit func(line string, record Record)) error {
	return withLedgerFile(path, func(f *os.File) error { return scanRecords(f, visit) })
}

// withLedgerFile opens the ledger for use and closes it, reporting a close
// failure when use succeeded. A missing ledger is not an error.
func withLedgerFile(path string, use func(*os.File) error) (err error) {
	f, err := os.Open(path) //nolint:gosec // path is resolved below .strategist/memory
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("leveling: open role level ledger: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("leveling: close role level ledger: %w", closeErr)
		}
	}()
	return use(f)
}

func scanRecords(r io.Reader, visit func(line string, record Record)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLedgerRecordBytes+1)
	for scanner.Scan() {
		if record, ok := parseLedgerLine(scanner.Bytes()); ok {
			visit(scanner.Text(), record)
		}
	}
	return ledgerReadError(scanner.Err())
}

// parseLedgerLine decodes one ledger line, rejecting malformed JSON and
// unsupported schema versions.
func parseLedgerLine(line []byte) (Record, bool) {
	var record Record
	if json.Unmarshal(line, &record) != nil || (record.SchemaVersion != "" && record.SchemaVersion != LedgerSchemaVersion) {
		return Record{}, false
	}
	if record.SchemaVersion == "" {
		record.SchemaVersion = LedgerSchemaVersion
	}
	// Legacy lines may carry a mixed-case role; normalize on read so
	// reporting, rotation keys and the mission view see one role.
	record.Role = NormalizeRole(record.Role)
	return record, true
}

func ledgerReadError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "token too long") {
		return fmt.Errorf("leveling_ledger_record_oversized: maximum record size is %d bytes", maxLedgerRecordBytes)
	}
	return fmt.Errorf("leveling: read role level ledger: %w", err)
}

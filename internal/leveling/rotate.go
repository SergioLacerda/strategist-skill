package leveling

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ledgerKey struct{ mission, role, run string }

// scanLedger calls visit for every well-formed line of the ledger, in order. A
// missing ledger is a no-op; malformed lines are skipped so a damaged history
// never blocks a report, a rotation or a mission.
func scanLedger(path string, visit func(line string, record Record)) error {
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
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) == nil {
			// Legacy lines may carry a mixed-case role; normalize on read so
			// reporting, rotation keys and the mission view see one role.
			record.Role = NormalizeRole(record.Role)
			visit(scanner.Text(), record)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	return nil
}

// RotateLedger compacts the ledger to at most maxRecords records when it holds
// more. The latest tuple of every mission, role and run is always kept, because
// that is what reuse reads, even when that exceeds maxRecords; remaining room is
// filled with the most recent older records. It returns how many records were
// dropped. A missing ledger is a no-op; malformed lines are dropped by a
// rotation. The rewrite is atomic.
func RotateLedger(path string, maxRecords int) (int, error) {
	var lines []string
	var records []Record
	err := scanLedger(path, func(line string, record Record) {
		lines, records = append(lines, line), append(records, record)
	})
	if err != nil || len(records) <= maxRecords {
		return 0, err
	}
	keep, kept := selectRecordsToKeep(records, maxRecords)
	if kept == len(records) {
		return 0, nil
	}
	if err := writeLedgerLines(path, lines, keep); err != nil {
		return 0, err
	}
	return len(records) - kept, nil
}

// selectRecordsToKeep marks the latest record of every key, then fills the
// remaining room up to maxRecords with the most recent older records.
func selectRecordsToKeep(records []Record, maxRecords int) (keep []bool, kept int) {
	keep = make([]bool, len(records))
	latest := map[ledgerKey]int{}
	for i, record := range records {
		latest[ledgerKey{record.MissionID, record.Role, record.Run}] = i
	}
	for _, i := range latest {
		keep[i] = true
		kept++
	}
	for i := len(records) - 1; i >= 0 && kept < maxRecords; i-- {
		if !keep[i] {
			keep[i] = true
			kept++
		}
	}
	return keep, kept
}

func writeLedgerLines(path string, lines []string, keep []bool) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".role-levels-*.tmp")
	if err != nil {
		return fmt.Errorf("leveling: create rotation file: %w", err)
	}
	name := tmp.Name()
	if err := writeKeptLines(tmp, lines, keep); err != nil {
		return errors.Join(err, os.Remove(name))
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return errors.Join(fmt.Errorf("leveling: chmod rotation file: %w", err), os.Remove(name))
	}
	if err := os.Rename(name, path); err != nil {
		return errors.Join(fmt.Errorf("leveling: replace role level ledger: %w", err), os.Remove(name))
	}
	return nil
}

// writeKeptLines writes the kept lines and closes the file, reporting the first
// failure.
func writeKeptLines(f *os.File, lines []string, keep []bool) error {
	w := bufio.NewWriter(f)
	var writeErr error
	for i, line := range lines {
		if !keep[i] || writeErr != nil {
			continue
		}
		_, writeErr = w.WriteString(line + "\n")
	}
	if writeErr == nil {
		writeErr = w.Flush()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return errors.Join(fmt.Errorf("leveling: write rotation file: %w", writeErr), closeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("leveling: close rotation file: %w", closeErr)
	}
	return nil
}

// RotateLedgerIfLarge rotates only when the ledger file is larger than
// maxBytes, so the common append path costs a single stat, not a read.
func RotateLedgerIfLarge(path string, maxBytes int64, maxRecords int) (int, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("leveling: stat role level ledger: %w", err)
	}
	if info.Size() <= maxBytes {
		return 0, nil
	}
	return RotateLedger(path, maxRecords)
}

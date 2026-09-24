package leveling

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ledgerKey struct{ mission, role, run string }

// RotateLedger compacts the ledger to at most maxRecords records when it holds
// more. The latest tuple of every mission, role and run is always kept, because
// that is what reuse reads, even when that exceeds maxRecords; remaining room is
// filled with the most recent older records. It returns how many records were
// dropped. A missing ledger is a no-op; malformed lines are dropped by a
// rotation. The rewrite is atomic.
func RotateLedger(path string, maxRecords int) (int, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return 0, fmt.Errorf("leveling: create ledger directory: %w", err)
	}
	var dropped int
	err := withLedgerLock(path, func() (lockErr error) {
		dropped, lockErr = rotateLocked(path, maxRecords)
		return lockErr
	})
	return dropped, err
}

// rotateLocked compacts the ledger while the caller holds the ledger lock.
func rotateLocked(path string, maxRecords int) (int, error) {
	var lines []string
	var records []Record
	err := scanLedgerUnlocked(path, func(line string, record Record) {
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
	writeErr := flushKeptLines(f, lines, keep)
	closeErr := f.Close()
	if writeErr != nil {
		return errors.Join(writeErr, closeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("leveling: close rotation file: %w", closeErr)
	}
	return nil
}

func flushKeptLines(f *os.File, lines []string, keep []bool) error {
	w := bufio.NewWriter(f)
	for i, line := range lines {
		if !keep[i] {
			continue
		}
		if _, err := w.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("leveling: write rotation file: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("leveling: write rotation file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("leveling: write rotation file: %w", err)
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

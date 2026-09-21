package leveling

import (
	"bufio"
	"encoding/json"
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
	lines, records, err := readLedgerLines(path)
	if err != nil || len(records) <= maxRecords {
		return 0, err
	}
	keep := make([]bool, len(records))
	latest := map[ledgerKey]int{}
	for i, record := range records {
		latest[ledgerKey{record.MissionID, record.Role, record.Run}] = i
	}
	kept := 0
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
	if kept == len(records) {
		return 0, nil
	}
	if err := writeLedgerLines(path, lines, keep); err != nil {
		return 0, err
	}
	return len(records) - kept, nil
}

func readLedgerLines(path string) (lines []string, records []Record, err error) {
	f, err := os.Open(path) //nolint:gosec // path is resolved below .strategist/memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("leveling: open role level ledger: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			lines, records, err = nil, nil, fmt.Errorf("leveling: close role level ledger: %w", closeErr)
		}
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) != nil {
			continue
		}
		lines = append(lines, scanner.Text())
		records = append(records, record)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, fmt.Errorf("leveling: read role level ledger: %w", scanErr)
	}
	return lines, records, nil
}

func writeLedgerLines(path string, lines []string, keep []bool) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".role-levels-*.tmp")
	if err != nil {
		return fmt.Errorf("leveling: create rotation file: %w", err)
	}
	tmpName := tmp.Name()
	w := bufio.NewWriter(tmp)
	for i, line := range lines {
		if !keep[i] {
			continue
		}
		if _, err := w.WriteString(line + "\n"); err != nil {
			return errors.Join(fmt.Errorf("leveling: write rotation file: %w", err), tmp.Close(), os.Remove(tmpName))
		}
	}
	if err := w.Flush(); err != nil {
		return errors.Join(fmt.Errorf("leveling: flush rotation file: %w", err), tmp.Close(), os.Remove(tmpName))
	}
	if err := tmp.Close(); err != nil {
		return errors.Join(fmt.Errorf("leveling: close rotation file: %w", err), os.Remove(tmpName))
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return errors.Join(fmt.Errorf("leveling: chmod rotation file: %w", err), os.Remove(tmpName))
	}
	if err := os.Rename(tmpName, path); err != nil {
		return errors.Join(fmt.Errorf("leveling: replace role level ledger: %w", err), os.Remove(tmpName))
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

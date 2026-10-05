package leveling

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RotateBudgetEvents keeps the most recent valid budget events.
func RotateBudgetEvents(path string, maxRecords int) (int, error) {
	if maxRecords < 1 {
		return 0, fmt.Errorf("leveling: budget ledger max records must be positive")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return 0, fmt.Errorf("leveling: create budget ledger directory: %w", err)
	}
	var dropped int
	err := withLedgerLock(path, func() error {
		records, err := readBudgetEventsUnlocked(path)
		if err != nil || len(records) <= maxRecords {
			return err
		}
		dropped = len(records) - maxRecords
		return writeBudgetEventsUnlocked(path, records[dropped:])
	})
	return dropped, err
}

func writeBudgetEventsUnlocked(path string, records []BudgetEventRecord) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".budget-events-*.tmp")
	if err != nil {
		return fmt.Errorf("leveling: create budget rotation file: %w", err)
	}
	name := tmp.Name()
	encoder := bufio.NewWriter(tmp)
	if err := encodeBudgetEvents(encoder, records); err != nil {
		cleanupBudgetRotation(tmp, name)
		return err
	}
	if err := encoder.Flush(); err != nil {
		cleanupBudgetRotation(tmp, name)
		return fmt.Errorf("leveling: flush budget rotation file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name) //nolint:errcheck // cleanup follows a close failure
		return fmt.Errorf("leveling: close budget rotation file: %w", err)
	}
	if err := os.Chmod(name, 0o600); err != nil {
		_ = os.Remove(name) //nolint:errcheck // cleanup follows a chmod failure
		return fmt.Errorf("leveling: chmod budget rotation file: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name) //nolint:errcheck // cleanup follows a rename failure
		return fmt.Errorf("leveling: replace budget ledger: %w", err)
	}
	return nil
}

func encodeBudgetEvents(encoder *bufio.Writer, records []BudgetEventRecord) error {
	for _, record := range records {
		encoded, marshalErr := json.Marshal(record)
		if marshalErr != nil {
			return fmt.Errorf("leveling: encode budget rotation event: %w", marshalErr)
		}
		if _, err := encoder.Write(append(encoded, '\n')); err != nil {
			return fmt.Errorf("leveling: write budget rotation file: %w", err)
		}
	}
	return nil
}

func cleanupBudgetRotation(tmp *os.File, name string) {
	_ = tmp.Close()     //nolint:errcheck // cleanup follows an earlier rotation failure
	_ = os.Remove(name) //nolint:errcheck // cleanup follows an earlier rotation failure
}

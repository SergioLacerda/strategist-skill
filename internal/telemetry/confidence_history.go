package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppendConfidenceRecord writes one validated JSONL record to runtime memory.
func AppendConfidenceRecord(path string, record ConfidenceRecord) error {
	_, err := AppendConfidenceRecordOnce(path, record)
	return err
}

// AppendConfidenceRecordOnce appends a record exactly once.
func AppendConfidenceRecordOnce(path string, record ConfidenceRecord) (appended bool, err error) {
	if err := ValidateConfidenceRecord(record); err != nil {
		return false, err
	}
	if record.EventID == "" {
		record.EventID = ConfidenceEventID(record)
	}
	if err := prepareConfidenceHistory(path); err != nil {
		return false, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644) //nolint:gosec // path is owned by runtime memory
	if err != nil {
		return false, fmt.Errorf("confidence record: open history: %w", err)
	}
	defer closeFileWithContext(f, &err, "confidence record: close history")
	if err := lockFileExclusive(f); err != nil {
		return false, fmt.Errorf("confidence record: lock history: %w", err)
	}
	defer unlockConfidenceFile(f, &err)
	return appendConfidenceRecordLocked(path, f, record)
}

func prepareConfidenceHistory(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("confidence record: create parent: %w", err)
	}
	return nil
}

func appendConfidenceRecordLocked(path string, f *os.File, record ConfidenceRecord) (bool, error) {
	if duplicate, err := confidenceEventExists(path, record.EventID); err != nil {
		return false, err
	} else if duplicate {
		return false, nil
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("confidence record: encode: %w", err)
	}
	if _, err := fmt.Fprintln(f, string(encoded)); err != nil {
		return false, fmt.Errorf("confidence record: append: %w", err)
	}
	return true, nil
}

func confidenceEventExists(path, eventID string) (exists bool, err error) {
	f, err := os.Open(path) //nolint:gosec // path is owned by runtime memory
	if err != nil {
		return false, fmt.Errorf("confidence record: inspect history: %w", err)
	}
	defer closeFileWithContext(f, &err, "confidence record: close inspected history")
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		var record ConfidenceRecord
		if json.Unmarshal(scanner.Bytes(), &record) == nil && record.EventID == eventID {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("confidence record: inspect history: %w", err)
	}
	return false, nil
}

func unlockConfidenceFile(f *os.File, err *error) {
	if unlockErr := unlockFile(f); unlockErr != nil && *err == nil {
		*err = fmt.Errorf("confidence record: unlock history: %w", unlockErr)
	}
}

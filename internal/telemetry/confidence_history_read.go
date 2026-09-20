package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ReadConfidenceRecords tolerates a missing history and malformed legacy lines.
func ReadConfidenceRecords(path string) ([]ConfidenceRecord, error) {
	records, _, err := ReadConfidenceRecordsWithDiagnostics(path)
	return records, err
}

// ReadConfidenceRecordsWithDiagnostics reads valid records and returns
// observable malformed, invalid, and duplicate counts.
func ReadConfidenceRecordsWithDiagnostics(path string) (records []ConfidenceRecord, diagnostics ConfidenceReadDiagnostics, err error) {
	f, err := os.Open(path) //nolint:gosec // path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, ConfidenceReadDiagnostics{}, nil
	}
	if err != nil {
		return nil, ConfidenceReadDiagnostics{}, fmt.Errorf("open confidence history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close confidence history")
	if err := lockFile(f); err != nil {
		return nil, ConfidenceReadDiagnostics{}, fmt.Errorf("lock confidence history: %w", err)
	}
	defer unlockConfidenceFile(f, &err)
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if record, ok := parseConfidenceRecordLine(scanner.Bytes(), seen, &diagnostics); ok {
			records = append(records, record)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, diagnostics, fmt.Errorf("scan confidence history: %w", err)
	}
	return records, diagnostics, nil
}

func parseConfidenceRecordLine(line []byte, seen map[string]struct{}, diagnostics *ConfidenceReadDiagnostics) (ConfidenceRecord, bool) {
	var record ConfidenceRecord
	if len(line) == 0 {
		return record, false
	}
	if err := json.Unmarshal(line, &record); err != nil {
		recordParseFailure(diagnostics, true, err)
		return record, false
	}
	record = normalizeConfidenceRecord(record)
	if err := ValidateConfidenceRecord(record); err != nil {
		recordParseFailure(diagnostics, false, err)
		return record, false
	}
	if confidenceRecordDuplicate(record, seen) {
		diagnostics.DuplicateEvents++
		return record, false
	}
	return record, true
}

func recordParseFailure(diagnostics *ConfidenceReadDiagnostics, malformed bool, err error) {
	if malformed {
		diagnostics.MalformedLines++
	} else {
		diagnostics.InvalidRecords++
	}
	diagnostics.Reasons = append(diagnostics.Reasons, err.Error())
}

package initiative

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ReadRecords reads and validates every line of the ledger at path,
// skipping malformed or invalid lines. A missing file yields no records
// and no error.
func ReadRecords(path string) (records []Record, err error) {
	f, openErr := os.Open(path) //nolint:gosec // caller resolves the runtime memory path
	if errors.Is(openErr, os.ErrNotExist) {
		return nil, nil
	}
	if openErr != nil {
		return nil, fmt.Errorf("initiative: open ledger: %w", openErr)
	}
	defer closeLedger(f, &err)
	return scanRecords(f)
}

func closeLedger(f *os.File, err *error) {
	if cerr := f.Close(); cerr != nil && *err == nil {
		*err = fmt.Errorf("initiative: close ledger: %w", cerr)
	}
}

func scanRecords(f *os.File) ([]Record, error) {
	var records []Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxLedgerRecordBytes+1)
	for scanner.Scan() {
		if record, ok := parseRecordLine(scanner.Bytes()); ok {
			records = append(records, record)
		}
	}
	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "token too long") {
			return nil, fmt.Errorf("initiative_ledger_record_oversized: maximum record size is %d bytes", maxLedgerRecordBytes)
		}
		return nil, fmt.Errorf("initiative: read ledger: %w", err)
	}
	return records, nil
}

func parseRecordLine(line []byte) (Record, bool) {
	var record Record
	if json.Unmarshal(line, &record) != nil || (record.SchemaVersion != "" && record.SchemaVersion != LedgerSchemaVersion) || validateRecord(record) != nil {
		return Record{}, false
	}
	if record.SchemaVersion == "" {
		record.SchemaVersion = LedgerSchemaVersion
	}
	record.Role = normalizeRole(record.Role)
	record.Supersedes = strings.TrimSpace(record.Supersedes)
	return record, true
}

// LatestAdvice returns the most recent advice Record in the ledger at path
// matching missionID, role, and runID, and whether one was found.
func LatestAdvice(path, missionID, role, runID string) (Record, bool, error) {
	records, err := ReadRecords(path)
	if err != nil {
		return Record{}, false, err
	}
	var latest Record
	found := false
	for _, record := range records {
		if record.Kind == RecordKindAdvice && record.MissionID == missionID && record.Role == normalizeRole(role) && record.RunID == runID {
			latest, found = record, true
		}
	}
	return latest, found, nil
}

// LatestResult returns the current result for an advice identity.
func LatestResult(path, missionID, role, runID, adviceID string) (Result, bool, error) {
	records, err := ReadRecords(path)
	if err != nil {
		return Result{}, false, err
	}
	var latest Result
	found := false
	for _, record := range records {
		if !resultRecordFor(record, missionID, role, runID, adviceID) {
			continue
		}
		if !found || record.Result.Sequence > latest.Sequence {
			latest, found = *record.Result, true
		}
	}
	return latest, found, nil
}

// LatestAssessment returns the derived PRECISE-SHOT assessment attached to
// the latest persisted result revision. Legacy result records return false.
func LatestAssessment(path, missionID, role, runID, adviceID string) (ResultAssessment, bool, error) {
	records, err := ReadRecords(path)
	if err != nil {
		return ResultAssessment{}, false, err
	}
	var latest ResultAssessment
	found := false
	sequence := 0
	for _, record := range records {
		if !assessmentRecordFor(record, missionID, role, runID, adviceID) {
			continue
		}
		if !found || record.Result.Sequence > sequence {
			latest, found, sequence = *record.Assessment, true, record.Result.Sequence
		}
	}
	return latest, found, nil
}

// resultRecordFor reports whether record is a result of the given advice identity.
func resultRecordFor(record Record, missionID, role, runID, adviceID string) bool {
	return record.Kind == RecordKindResult && record.Result != nil && record.MissionID == missionID &&
		record.Role == normalizeRole(role) && record.RunID == runID && record.AdviceID == adviceID
}

func assessmentRecordFor(record Record, missionID, role, runID, adviceID string) bool {
	return record.Assessment != nil && resultRecordFor(record, missionID, role, runID, adviceID)
}

package initiative

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RecordKindAdvice marks a ledger Record as carrying an Advice.
const RecordKindAdvice = "advice"

// RecordKindResult marks a ledger Record as carrying a Result.
const RecordKindResult = "result"

// Record is a single line of the append-only INITIATIVE ledger, holding
// either an Advice or a Result.
type Record struct {
	Kind       string  `json:"kind"`
	MissionID  string  `json:"mission_id"`
	Role       string  `json:"role"`
	RunID      string  `json:"run_id"`
	AdviceID   string  `json:"advice_id"`
	Supersedes string  `json:"supersedes,omitempty"`
	Timestamp  string  `json:"timestamp"`
	Advice     *Advice `json:"advice,omitempty"`
	Result     *Result `json:"result,omitempty"`
}

// AppendRecord validates record and appends it as a JSON line to the ledger
// file at path, creating the file and its parent directory if needed.
func AppendRecord(path string, record Record) error {
	if err := validateRecord(record); err != nil {
		return err
	}
	if record.Kind != RecordKindAdvice {
		return fmt.Errorf("initiative_record_invalid: result records must be appended by the validated runtime")
	}
	return withLedgerLock(path, func() error { return appendRecordUnlocked(path, record) })
}

func appendRecordUnlocked(path string, record Record) error {
	if err := validateRecord(record); err != nil {
		return err
	}
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("initiative: encode record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("initiative: create ledger directory: %w", err)
	}
	return writeLedgerLine(path, encoded)
}

func writeLedgerLine(path string, encoded []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // caller resolves the runtime memory path
	if err != nil {
		return fmt.Errorf("initiative: open ledger: %w", err)
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return errors.Join(fmt.Errorf("initiative: write ledger: %w", err), f.Close())
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("initiative: close ledger: %w", err)
	}
	return nil
}

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
	for scanner.Scan() {
		if record, ok := parseRecordLine(scanner.Bytes()); ok {
			records = append(records, record)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("initiative: read ledger: %w", err)
	}
	return records, nil
}

func parseRecordLine(line []byte) (Record, bool) {
	var record Record
	if json.Unmarshal(line, &record) != nil || validateRecord(record) != nil {
		return Record{}, false
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

func validateRecord(record Record) error {
	if !validRecordKind(record.Kind) {
		return fmt.Errorf("initiative_record_invalid: unknown kind %q", record.Kind)
	}
	if !hasRecordIdentity(record) {
		return fmt.Errorf("initiative_record_invalid: mission, role, run, and advice identity are required")
	}
	if record.Kind == RecordKindAdvice {
		return validateAdviceRecord(record)
	}
	return validateResultRecord(record)
}

func validRecordKind(kind string) bool {
	return kind == RecordKindAdvice || kind == RecordKindResult
}

func hasRecordIdentity(record Record) bool {
	return record.MissionID != "" && normalizeRole(record.Role) != "" &&
		record.RunID != "" && record.AdviceID != ""
}

func validateAdviceRecord(record Record) error {
	if record.Advice == nil || record.Result != nil || record.Advice.AdviceID != record.AdviceID {
		return fmt.Errorf("initiative_record_invalid: advice record payload mismatch")
	}
	return record.Advice.Validate()
}

func validateResultRecord(record Record) error {
	if record.Result == nil || record.Advice != nil || record.Result.AdviceID != record.AdviceID {
		return fmt.Errorf("initiative_record_invalid: result record payload mismatch")
	}
	return nil
}

package initiative

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RecordKindAdvice marks a ledger Record as carrying an Advice.
const RecordKindAdvice = "advice"

// RecordKindResult marks a ledger Record as carrying a Result.
const RecordKindResult = "result"

// LedgerSchemaVersion is the version emitted by new INITIATIVE records.
const LedgerSchemaVersion = "1"

const maxLedgerRecordBytes = 1 << 20

// Record is a single line of the append-only INITIATIVE ledger, holding
// either an Advice or a Result.
type Record struct {
	SchemaVersion string            `json:"schema_version,omitempty"`
	Kind          string            `json:"kind"`
	MissionID     string            `json:"mission_id"`
	Role          string            `json:"role"`
	RunID         string            `json:"run_id"`
	AdviceID      string            `json:"advice_id"`
	Supersedes    string            `json:"supersedes,omitempty"`
	Timestamp     string            `json:"timestamp"`
	Advice        *Advice           `json:"advice,omitempty"`
	Result        *Result           `json:"result,omitempty"`
	Assessment    *ResultAssessment `json:"assessment,omitempty"`
}

// AppendRecord validates record and appends it as a JSON line to the ledger
// file at path, creating the file and its parent directory if needed.
func AppendRecord(path string, record Record) error {
	if record.Kind != RecordKindAdvice {
		return fmt.Errorf("initiative_record_invalid: result records must be appended by the validated runtime")
	}
	if err := validateRecord(record); err != nil {
		return err
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
	if record.SchemaVersion == "" {
		record.SchemaVersion = LedgerSchemaVersion
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

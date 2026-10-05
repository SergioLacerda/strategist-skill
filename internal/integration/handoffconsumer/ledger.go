package handoffconsumer

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Record is one line of the integration call ledger: metadata about a pre-check,
// never artifact content, a provider body or a credential. It is the call budget's
// source of truth and the ground truth the pilot measures against.
type Record struct {
	At            string  `json:"at"`
	MissionID     string  `json:"mission_id"`
	Transition    string  `json:"transition"`
	Subject       string  `json:"subject"`
	PreferredPath string  `json:"preferred_path"`
	EffectivePath string  `json:"effective_path"`
	Reason        string  `json:"reason,omitempty"`
	FellBack      bool    `json:"fell_back"`
	Suspect       bool    `json:"suspect,omitempty"`
	Binding       string  `json:"binding"`
	Model         string  `json:"model,omitempty"`
	Attempted     bool    `json:"attempted"`
	Approved      bool    `json:"approved"`
	Confidence    float64 `json:"confidence,omitempty"`
	Threshold     float64 `json:"threshold,omitempty"`
	InputTokens   int     `json:"input_tokens,omitempty"`
	OutputTokens  int     `json:"output_tokens,omitempty"`
	LatencyMS     int64   `json:"latency_ms,omitempty"`
	Provider      string  `json:"provider,omitempty"`
	// Answered is true when the provider returned a valid answer (approved or
	// signal); a fallback has none and is never reused.
	Answered bool `json:"answered,omitempty"`
	// Hints are the unsatisfied criterion keys of a signal, kept for reuse.
	Hints []string `json:"hints,omitempty"`
	// Reused marks a record that reports a reuse of an earlier answer, not a call.
	Reused bool `json:"reused,omitempty"`
	// CalibrationID is the pinned calibration at the time of the call. Calibrated
	// is true only when that calibration was valid and decided the threshold;
	// otherwise CalibrationReason says why the configured default applied.
	CalibrationID     string `json:"calibration_id,omitempty"`
	Calibrated        bool   `json:"calibrated,omitempty"`
	CalibrationReason string `json:"calibration_reason,omitempty"`
	// Integrity is the digest of every other field, so an edited record is not trusted.
	Integrity string `json:"integrity,omitempty"`
}

func (r Record) digest() (string, error) {
	r.Integrity = ""
	raw, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("encode ledger record: %w", err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}

// Ledger is an append-only JSONL file.
type Ledger struct{ Path string }

// Append writes one record with owner-only permissions.
func (l Ledger) Append(record Record) error {
	digest, err := record.digest()
	if err != nil {
		return err
	}
	record.Integrity = digest
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode ledger record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o750); err != nil {
		return fmt.Errorf("create ledger directory: %w", err)
	}
	file, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: ledger path is owned by runtime memory
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return errors.Join(fmt.Errorf("write ledger: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close ledger: %w", err)
	}
	return nil
}

// Count returns how many provider calls were attempted for a mission and
// transition. A missing ledger is zero; malformed lines are ignored.
func (l Ledger) Count(missionID, transition string) (int, error) {
	file, err := os.Open(l.Path) //nolint:gosec // G304: ledger path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("open ledger: %w", err)
	}
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) == nil && record.Attempted && record.MissionID == missionID && record.Transition == transition {
			count++
		}
	}
	return count, errors.Join(scanner.Err(), file.Close())
}

// scanIntact calls visit for every record whose integrity digest matches; an
// edited or malformed line is skipped. A missing ledger visits nothing.
func (l Ledger) scanIntact(visit func(Record)) error {
	file, err := os.Open(l.Path) //nolint:gosec // G304: ledger path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if record, ok := intactRecord(scanner.Bytes()); ok {
			visit(record)
		}
	}
	return errors.Join(scanner.Err(), file.Close())
}

func intactRecord(line []byte) (Record, bool) {
	var record Record
	if json.Unmarshal(line, &record) != nil {
		return Record{}, false
	}
	digest, err := record.digest()
	return record, err == nil && digest == record.Integrity
}

// Latest returns the most recent intact, answered record of a mission,
// transition and subject, ignoring reuse markers.
func (l Ledger) Latest(missionID, transition, subject string) (Record, bool, error) {
	var latest Record
	found := false
	err := l.scanIntact(func(record Record) {
		if record.Answered && !record.Reused && record.MissionID == missionID && record.Transition == transition && record.Subject == subject {
			latest, found = record, true
		}
	})
	return latest, found, err
}

// LatestRecord returns the most recent intact record of a mission and
// transition, answered or not, ignoring reuse markers. It is for read-only
// reporting.
func (l Ledger) LatestRecord(missionID, transition string) (Record, bool, error) {
	var latest Record
	found := false
	err := l.scanIntact(func(record Record) {
		if !record.Reused && record.MissionID == missionID && record.Transition == transition {
			latest, found = record, true
		}
	})
	return latest, found, err
}

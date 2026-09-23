package leveling

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

// Record is one persisted level tuple, appended to the role-level ledger so a
// phase reuses one level and telemetry can group by it. Reason is set when a
// new tuple supersedes an earlier one (for example `escalated`).
type Record struct {
	MissionID string `json:"mission_id"`
	// Run distinguishes repeated executions of the same role in one mission (for
	// example an Archivist revision loop); empty is the default run.
	Run string `json:"run,omitempty"`
	Level
	Reason    string `json:"reason,omitempty"`
	Timestamp string `json:"timestamp"`
}

// NormalizeRole is the canonical spelling of a role id in the ledger. Every
// read and write goes through it, so `Ranger` and `ranger` are one role for
// reuse, reporting and the mission view instead of three separate histories.
func NormalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

// AppendRecord appends a record as one JSON line, stamping the time when unset.
func AppendRecord(path string, record Record) error {
	record.Role = NormalizeRole(record.Role)
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("leveling: encode role level record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("leveling: create ledger directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // path is resolved below .strategist/memory
	if err != nil {
		return fmt.Errorf("leveling: open role level ledger: %w", err)
	}
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return errors.Join(fmt.Errorf("leveling: write role level record: %w", err), f.Close())
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("leveling: close role level ledger: %w", err)
	}
	return nil
}

// LatestRecord returns the most recent record of the default run for a mission
// and role.
func LatestRecord(path, missionID, role string) (Record, bool, error) {
	return LatestRunRecord(path, missionID, role, "")
}

// LatestRunRecord returns the most recent record for a mission, role and run. A
// missing ledger or no match reports ok=false; malformed lines are skipped so a
// damaged history never blocks a mission.
func LatestRunRecord(path, missionID, role, run string) (latest Record, found bool, err error) {
	f, err := openLedger(path)
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	defer func() {
		closeLatestRecordLedger(f, &latest, &found, &err)
	}()
	return scanLatestRecord(f, missionID, role, run)
}

func openLedger(path string) (*os.File, error) {
	f, err := os.Open(path) //nolint:gosec // path is resolved below .strategist/memory
	if os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, fmt.Errorf("leveling: open role level ledger: %w", err)
	}
	return f, nil
}

func scanLatestRecord(f *os.File, missionID, role, run string) (Record, bool, error) {
	scanner := bufio.NewScanner(f)
	var latest Record
	found := false
	for scanner.Scan() {
		latest, found = updateLatestRecord(scanner.Bytes(), missionID, role, run, latest, found)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return Record{}, false, fmt.Errorf("leveling: read role level ledger: %w", scanErr)
	}
	return latest, found, nil
}

func updateLatestRecord(raw []byte, missionID, role, run string, latest Record, found bool) (Record, bool) {
	var record Record
	if json.Unmarshal(raw, &record) != nil {
		return latest, found
	}
	record.Role = NormalizeRole(record.Role)
	if record.MissionID != missionID || record.Role != NormalizeRole(role) || record.Run != run {
		return latest, found
	}
	return record, true
}

func closeLatestRecordLedger(f *os.File, latest *Record, found *bool, err *error) {
	if closeErr := f.Close(); closeErr != nil && *err == nil {
		*latest, *found, *err = Record{}, false, fmt.Errorf("leveling: close role level ledger: %w", closeErr)
	}
}

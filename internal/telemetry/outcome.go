package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// OutcomeEntry is the canonical JSON structure written to outcomes.tmp per mission.
type OutcomeEntry struct {
	MissionID string           `json:"mission_id"`
	Status    string           `json:"status"`
	Timestamp string           `json:"timestamp"`
	JewelIDs  []string         `json:"jewel_ids,omitempty"`
	Gates     []GateAuditEntry `json:"gates,omitempty"`
}

// GateAuditEntry records one gate approval event.
type GateAuditEntry struct {
	Type       string `json:"type"`
	ApprovedAt string `json:"approved_at"`
	Response   string `json:"response"`
}

// ValidateOutcomeLine parses a single JSON line and checks required fields.
func ValidateOutcomeLine(line string) error {
	var e OutcomeEntry
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return fmt.Errorf("outcome line is not valid JSON: %w", err)
	}
	var errs []error
	if e.MissionID == "" {
		errs = append(errs, errors.New("mission_id is required"))
	}
	if e.Status == "" {
		errs = append(errs, errors.New("status is required"))
	}
	if e.Timestamp == "" {
		errs = append(errs, errors.New("timestamp is required"))
	}
	return errors.Join(errs...)
}

// AppendOutcomeLine validates line and appends it with a newline to path,
// unless an entry with the same mission_id already exists in path — the
// idempotency key for outcomes.jsonl/outcomes.tmp (ADR-0004: the buffer
// flush and a direct write can target the same mission_id and duplicate it).
// If validation fails the line is not written and the error is returned.
// appended reports whether the line was written (false means a duplicate
// mission_id was skipped). The file is created if absent. A shared flock is
// held during the read-then-write so concurrent appenders remain compatible
// while a flush's exclusive lock blocks new appends until the cat+truncate
// sequence completes.
func AppendOutcomeLine(path, line string) (appended bool, err error) { //nolint:dupl // mirrors route-decision append semantics for a separate schema
	if err = ValidateOutcomeLine(line); err != nil {
		return false, fmt.Errorf("outcome validation failed: %w", err)
	}
	var entry OutcomeEntry
	if err = json.Unmarshal([]byte(line), &entry); err != nil {
		return false, fmt.Errorf("outcome line is not valid JSON: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // G304: outcomes path is owned by the Strategist runtime memory domain
	if err != nil {
		return false, fmt.Errorf("open outcomes file: %w", err)
	}
	defer closeFileWithContext(f, &err, "close outcomes file")
	return appendOutcomeLineLocked(f, entry.MissionID, line)
}

func appendOutcomeLineLocked(f *os.File, missionID, line string) (appended bool, err error) {
	if err = lockFile(f); err != nil {
		return false, fmt.Errorf("lock outcomes file: %w", err)
	}
	defer unlockOutcomeFile(f, &err)

	exists, err := missionIDExists(f, missionID)
	if err != nil {
		return false, fmt.Errorf("scan outcomes file: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return false, fmt.Errorf("seek outcomes file: %w", err)
	}
	if _, err = fmt.Fprintln(f, line); err != nil {
		return false, fmt.Errorf("write outcome line: %w", err)
	}
	return true, nil
}

// missionIDExists scans f for an existing outcome entry whose mission_id
// matches. Lines that fail to parse are tolerated and skipped rather than
// treated as an error, since outcomes.jsonl is append-only historical data
// that may include entries from schema revisions.
func missionIDExists(f *os.File, missionID string) (bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek outcomes file start: %w", err)
	}
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		if outcomeLineHasMissionID(scanner.Bytes(), missionID) {
			return true, nil
		}
	}
	return false, jsonlScannerErr(scanner, "scan outcomes file")
}

func outcomeLineHasMissionID(line []byte, missionID string) bool {
	if len(line) == 0 {
		return false
	}
	var entry OutcomeEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return false
	}
	return entry.MissionID == missionID
}

func unlockOutcomeFile(f *os.File, err *error) {
	if unlockErr := unlockFile(f); unlockErr != nil && *err == nil {
		*err = fmt.Errorf("unlock outcomes file: %w", unlockErr)
	}
}

// FlushOutcomeBuffer, readOutcomeBuffer, flushOutcomeBufferData,
// appendBufferedOutcomeLine, and truncateOutcomeBuffer live in
// outcome_flush.go, split out to keep this file under the repo's file-size
// budget.

// AppendOutcomeLineSafe calls AppendOutcomeLine and logs errors without propagating them.
// Use this at all call sites where learning failures must not block the mission result.
func AppendOutcomeLineSafe(path, line string) {
	appended, err := AppendOutcomeLine(path, line)
	if err != nil {
		slog.Warn("outcome write failed (non-blocking)", "error", err)
		return
	}
	if !appended {
		slog.Debug("outcome write skipped: duplicate mission_id", "path", path)
	}
}

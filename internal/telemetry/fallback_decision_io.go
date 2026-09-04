package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// AppendFallbackDecisionLine validates line and appends it with a newline to
// path, unless an entry with the same mission_id+slot already exists in
// path. The compound key (rather than RouteDecision's mission_id-only key)
// reflects that a single mission has up to three independent slots, each of
// which can degrade separately. If validation fails the line is not written
// and the error is returned. appended reports whether the line was written
// (false means a duplicate mission_id+slot was skipped). The file is created
// if absent.
func AppendFallbackDecisionLine(path, line string) (appended bool, err error) { //nolint:dupl // mirrors route decision append semantics for a separate schema
	if err = ValidateFallbackDecisionLine(line); err != nil {
		return false, fmt.Errorf("fallback decision validation failed: %w", err)
	}
	var entry FallbackDecision
	if err = json.Unmarshal([]byte(line), &entry); err != nil {
		return false, fmt.Errorf("fallback decision line is not valid JSON: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // G304: fallback decision path is owned by runtime memory
	if err != nil {
		return false, fmt.Errorf("open fallback decisions file: %w", err)
	}
	defer closeFileWithContext(f, &err, "close fallback decisions file")
	return appendFallbackDecisionLineLocked(f, entry.MissionID, entry.Slot, line)
}

func appendFallbackDecisionLineLocked(f *os.File, missionID, slot, line string) (appended bool, err error) {
	if err = lockFile(f); err != nil {
		return false, fmt.Errorf("lock fallback decisions file: %w", err)
	}
	defer unlockFallbackDecisionFile(f, &err)

	exists, err := fallbackDecisionKeyExists(f, missionID, slot)
	if err != nil {
		return false, fmt.Errorf("scan fallback decisions file: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return false, fmt.Errorf("seek fallback decisions file: %w", err)
	}
	if _, err = fmt.Fprintln(f, line); err != nil {
		return false, fmt.Errorf("write fallback decision line: %w", err)
	}
	return true, nil
}

// fallbackDecisionKeyExists scans f for an existing fallback decision entry
// whose mission_id and slot both match. Lines that fail to parse are
// tolerated and skipped rather than treated as an error, consistent with
// routeDecisionMissionIDExists's treatment of append-only historical data.
func fallbackDecisionKeyExists(f *os.File, missionID, slot string) (bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("seek fallback decisions file start: %w", err)
	}
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		if fallbackDecisionLineHasKey(scanner.Bytes(), missionID, slot) {
			return true, nil
		}
	}
	return false, jsonlScannerErr(scanner, "scan fallback decisions file")
}

func fallbackDecisionLineHasKey(line []byte, missionID, slot string) bool {
	if len(line) == 0 {
		return false
	}
	var entry FallbackDecision
	if err := json.Unmarshal(line, &entry); err != nil {
		return false
	}
	return entry.MissionID == missionID && entry.Slot == slot
}

func unlockFallbackDecisionFile(f *os.File, err *error) {
	if unlockErr := unlockFile(f); unlockErr != nil && *err == nil {
		*err = fmt.Errorf("unlock fallback decisions file: %w", unlockErr)
	}
}

// ReadFallbackDecisions reads path's JSONL history. A missing file is not an
// error — it returns a nil slice, so a workspace where no fallback has ever
// been applied reports an empty sample rather than failing, same posture as
// ReadHandoffChallenges. Malformed lines are skipped rather than treated as
// an error, so one bad historical record does not disable the signal.
func ReadFallbackDecisions(path string) (records []FallbackDecision, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: fallback decision history path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open fallback decision history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close fallback decision history")

	return scanFallbackDecisions(f)
}

func scanFallbackDecisions(r io.Reader) ([]FallbackDecision, error) {
	var records []FallbackDecision
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if rec, ok := parseFallbackDecisionLine(scanner.Bytes()); ok {
			records = append(records, rec)
		}
	}
	return records, jsonlScannerErr(scanner, "scan fallback decision history")
}

func parseFallbackDecisionLine(line []byte) (rec FallbackDecision, ok bool) {
	if len(line) == 0 {
		return rec, false
	}
	if err := json.Unmarshal(line, &rec); err != nil {
		return rec, false
	}
	return rec, true
}

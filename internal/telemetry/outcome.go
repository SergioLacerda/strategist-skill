package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
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
func AppendOutcomeLine(path, line string) (appended bool, err error) {
	if err = ValidateOutcomeLine(line); err != nil {
		return false, fmt.Errorf("outcome validation failed: %w", err)
	}
	var entry OutcomeEntry
	if err = json.Unmarshal([]byte(line), &entry); err != nil {
		return false, fmt.Errorf("outcome line is not valid JSON: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
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

// FlushOutcomeBuffer moves buffered entries from bufferPath (outcomes.tmp)
// into outcomesPath (outcomes.jsonl), skipping any whose mission_id already
// exists in outcomesPath. It replaces the raw `cat >> ; truncate`
// flush_procedure with a version that cannot introduce a duplicate outcome
// record when a previous flush was interrupted between the append and the
// truncate (ADR-0004: "two paths to the same data can cause duplicates").
// A missing buffer file is not an error — it means the buffer is empty.
// The buffer is only cleared after every line has been processed, so a
// failure mid-flush leaves it intact for the next mission's retry.
// Malformed buffered lines are skipped (logged, non-blocking) rather than
// aborting the flush, consistent with the learning contract's non-blocking
// invariant.
func FlushOutcomeBuffer(bufferPath, outcomesPath string) (flushed int, err error) {
	data, err := readOutcomeBuffer(bufferPath)
	if err != nil {
		return 0, err
	}
	if data == nil {
		return 0, nil
	}

	flushed, err = flushOutcomeBufferData(data, outcomesPath)
	if err != nil {
		return flushed, err
	}
	return flushed, truncateOutcomeBuffer(bufferPath)
}

func readOutcomeBuffer(bufferPath string) ([]byte, error) {
	data, err := os.ReadFile(bufferPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read outcomes buffer: %w", err)
	}
	return data, nil
}

func flushOutcomeBufferData(data []byte, outcomesPath string) (int, error) {
	flushed := 0
	scanner := newJSONLScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if appendBufferedOutcomeLine(outcomesPath, scanner.Text()) {
			flushed++
		}
	}
	return flushed, jsonlScannerErr(scanner, "scan outcomes buffer")
}

func appendBufferedOutcomeLine(outcomesPath, line string) bool {
	if line == "" {
		return false
	}
	appended, err := AppendOutcomeLine(outcomesPath, line)
	if err != nil {
		slog.Warn("outcome flush: skipping invalid buffered line (non-blocking)", "error", err)
		return false
	}
	return appended
}

func truncateOutcomeBuffer(bufferPath string) error {
	if err := os.Truncate(bufferPath, 0); err != nil {
		return fmt.Errorf("truncate outcomes buffer: %w", err)
	}
	return nil
}

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

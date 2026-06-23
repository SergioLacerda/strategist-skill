package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

// OutcomeEntry is the canonical JSON structure written to outcomes.tmp per mission.
type OutcomeEntry struct {
	MissionID string           `json:"mission_id"`
	Status    string           `json:"status"`
	Timestamp string           `json:"timestamp"`
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

// AppendOutcomeLine validates line and appends it with a newline to path.
// If validation fails the line is not written and the error is returned.
// The file is created if absent. A shared flock is held during the write so
// concurrent appenders remain compatible while a flush's exclusive lock blocks
// new appends until the cat+truncate sequence completes.
func AppendOutcomeLine(path, line string) (err error) {
	if err = ValidateOutcomeLine(line); err != nil {
		return fmt.Errorf("outcome validation failed: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open outcomes file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close outcomes file: %w", cerr)
		}
	}()
	if err = lockFile(f); err != nil {
		return fmt.Errorf("lock outcomes file: %w", err)
	}
	defer func() {
		if unlockErr := unlockFile(f); unlockErr != nil && err == nil {
			err = fmt.Errorf("unlock outcomes file: %w", unlockErr)
		}
	}()
	if _, err = fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("write outcome line: %w", err)
	}
	return nil
}

// AppendOutcomeLineSafe calls AppendOutcomeLine and logs errors without propagating them.
// Use this at all call sites where learning failures must not block the mission result.
func AppendOutcomeLineSafe(path, line string) {
	if err := AppendOutcomeLine(path, line); err != nil {
		slog.Warn("outcome write failed (non-blocking)", "error", err)
	}
}

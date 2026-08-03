package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// HandoffChallengeHistoryRelPath is relative to the .strategist runtime root.
const HandoffChallengeHistoryRelPath = "memory/handoff-challenges.jsonl"

// ChallengeRecord is the JSONL-persisted record for one Handoff Challenge
// verification attempt — internal/handoff.Result's fields, plus enough
// context to aggregate across missions/attempts (mission_id, transition,
// attempt, timestamp), per
// .analysis/refined/20260803-handoff-challenge-extensions/design.md § Item
// 3. internal/handoff.Result itself is reused as-is for the in-memory
// verification result and is not modified; the fields here are mirrored,
// not imported, since internal/telemetry must not depend on internal/handoff,
// a peer package (see internal/domain/architecture_test.go's
// TestLateralIsolation).
type ChallengeRecord struct {
	MissionID  string `json:"mission_id"`
	Transition string `json:"transition"`
	Attempt    int    `json:"attempt"`
	Timestamp  string `json:"timestamp"`

	Status            string   `json:"status"`
	Passed            bool     `json:"passed"`
	MissingRefs       []string `json:"missing_refs,omitempty"`
	MissingChallenges []string `json:"missing_challenges,omitempty"`
	MisclassifiedRefs []string `json:"misclassified_refs,omitempty"`
	GateMismatch      bool     `json:"gate_mismatch"`
	CriticalFailures  int      `json:"critical_failures"`
}

// HandoffChallengeHistoryPath returns the default runtime memory path for
// Handoff Challenge history.
func HandoffChallengeHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(HandoffChallengeHistoryRelPath))
}

// ReadHandoffChallenges reads path's JSONL history. A missing file is not
// an error — it returns a nil slice, so a workspace where no Handoff
// Challenge has ever run reports an empty sample rather than failing, same
// posture as ReadRecentSniperMaterializations. Malformed lines are skipped
// rather than treated as an error, so one bad historical record does not
// disable the signal.
func ReadHandoffChallenges(path string) (records []ChallengeRecord, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: handoff challenge history path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open handoff challenge history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close handoff challenge history")

	return scanHandoffChallenges(f)
}

func scanHandoffChallenges(r io.Reader) ([]ChallengeRecord, error) {
	var records []ChallengeRecord
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if rec, ok := parseHandoffChallengeLine(scanner.Bytes()); ok {
			records = append(records, rec)
		}
	}
	return records, jsonlScannerErr(scanner, "scan handoff challenge history")
}

func parseHandoffChallengeLine(line []byte) (rec ChallengeRecord, ok bool) {
	if len(line) == 0 {
		return rec, false
	}
	if err := json.Unmarshal(line, &rec); err != nil {
		return rec, false
	}
	return rec, true
}

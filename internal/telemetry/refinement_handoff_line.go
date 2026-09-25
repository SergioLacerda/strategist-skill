package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// HandoffMetricsRelPath is the Archivist's per-refinement handoff-metrics log,
// relative to the .strategist runtime root (skill.yaml#handoff_metrics_log).
const HandoffMetricsRelPath = "memory/handoff-metrics.jsonl"

// HandoffMetricsPath returns the runtime memory path of the handoff-metrics log.
func HandoffMetricsPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(HandoffMetricsRelPath))
}

// RefinementHandoffLine is one line of the handoff-metrics log. Pointer fields
// are null when the value was not measured: the contract expects nulls during
// rollout, so an unmeasured value is never guessed or derived.
type RefinementHandoffLine struct {
	MissionID             string   `json:"mission_id"`
	DiscoveryTokens       *int64   `json:"discovery_tokens"`
	BriefTokens           *int64   `json:"brief_tokens"`
	BriefCompressionRatio *float64 `json:"brief_compression_ratio"`
	RefinementReopens     int      `json:"refinement_reopens"`
	EvidenceCoverageRatio *float64 `json:"evidence_coverage_ratio"`
	Model                 *string  `json:"model"`
	Effort                *string  `json:"effort"`
	LevelSource           *string  `json:"level_source"`
}

// Validate rejects a line that would corrupt the history.
func (l RefinementHandoffLine) Validate() error {
	switch {
	case l.MissionID == "":
		return fmt.Errorf("handoff metrics: mission_id is required")
	case negativeInt(l.DiscoveryTokens):
		return fmt.Errorf("handoff metrics: discovery_tokens must be >= 0")
	case negativeInt(l.BriefTokens):
		return fmt.Errorf("handoff metrics: brief_tokens must be >= 0")
	case l.RefinementReopens < 0:
		return fmt.Errorf("handoff metrics: refinement_reopens must be >= 0")
	case negativeFloat(l.BriefCompressionRatio):
		return fmt.Errorf("handoff metrics: brief_compression_ratio must be >= 0")
	case negativeFloat(l.EvidenceCoverageRatio):
		return fmt.Errorf("handoff metrics: evidence_coverage_ratio must be >= 0")
	}
	return nil
}

func negativeInt(v *int64) bool     { return v != nil && *v < 0 }
func negativeFloat(v *float64) bool { return v != nil && *v < 0 }

// AppendRefinementHandoffLine appends the line unless the mission already has
// one; it reports whether anything was written. The append is serialized with
// an exclusive lock so concurrent roles cannot interleave or duplicate lines.
func AppendRefinementHandoffLine(path string, line RefinementHandoffLine) (appended bool, err error) {
	if err = line.Validate(); err != nil {
		return false, err
	}
	encoded, err := json.Marshal(line)
	if err != nil {
		return false, fmt.Errorf("handoff metrics: encode line: %w", err)
	}
	f, err := openHandoffHistoryLocked(path)
	if err != nil {
		return false, err
	}
	defer releaseHandoffHistory(f, &err)
	return appendHandoffLineLocked(f, line.MissionID, encoded)
}

// openHandoffHistoryLocked opens (creating it and its directory) the history
// and takes the exclusive lock; the caller releases it with releaseHandoffHistory.
func openHandoffHistoryLocked(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // runtime memory directory
		return nil, fmt.Errorf("handoff metrics: create memory directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644) //nolint:gosec // G304: handoff metrics path is owned by runtime memory
	if err != nil {
		return nil, fmt.Errorf("handoff metrics: open history: %w", err)
	}
	if err := lockFileExclusive(f); err != nil {
		closeFileWithContext(f, &err, "close handoff metrics history")
		return nil, fmt.Errorf("handoff metrics: %w", err)
	}
	return f, nil
}

func releaseHandoffHistory(f *os.File, err *error) {
	if unlockErr := unlockFile(f); unlockErr != nil && *err == nil {
		*err = fmt.Errorf("handoff metrics: %w", unlockErr)
	}
	closeFileWithContext(f, err, "close handoff metrics history")
}

func appendHandoffLineLocked(f *os.File, missionID string, encoded []byte) (bool, error) {
	exists, err := handoffMissionRecorded(f, missionID)
	if err != nil || exists {
		return false, err
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return false, fmt.Errorf("handoff metrics: seek history end: %w", err)
	}
	if _, err = fmt.Fprintf(f, "%s\n", encoded); err != nil {
		return false, fmt.Errorf("handoff metrics: write line: %w", err)
	}
	return true, nil
}

// handoffMissionRecorded scans the history for the mission. A line that does
// not parse is skipped, like the other append-only histories in this package.
func handoffMissionRecorded(f *os.File, missionID string) (bool, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("handoff metrics: seek history start: %w", err)
	}
	scanner := newJSONLScanner(f)
	for scanner.Scan() {
		var entry struct {
			MissionID string `json:"mission_id"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) == nil && entry.MissionID == missionID {
			return true, nil
		}
	}
	return false, jsonlScannerErr(scanner, "handoff metrics: scan history")
}

package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// SniperMaterializationHistoryRelPath is relative to the .strategist runtime root.
	SniperMaterializationHistoryRelPath = "memory/sniper-materializations.jsonl"
	// SniperMaterializationWindow is ADR-0008's rolling F3 conflict-attribution window.
	SniperMaterializationWindow = 30 * 24 * time.Hour
)

// SniperMaterializationRecord records a documentation target materialized by Sniper.
type SniperMaterializationRecord struct {
	MissionID      string    `json:"mission_id"`
	BasePath       string    `json:"base_path"`
	TargetPath     string    `json:"target_path"`
	MaterializedAt time.Time `json:"materialized_at"`
}

// SniperMaterializationHistoryPath returns the default runtime memory path for Sniper materialization history.
func SniperMaterializationHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(SniperMaterializationHistoryRelPath))
}

// AppendSniperMaterialization appends rec as one JSONL record.
func AppendSniperMaterialization(path string, rec SniperMaterializationRecord) (err error) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return fmt.Errorf("create sniper materialization history dir: %w", mkdirErr)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open sniper materialization history: %w", err)
	}
	defer closeSniperMaterializationFile(f, &err)

	return writeSniperMaterializationLine(f, rec)
}

func writeSniperMaterializationLine(f *os.File, rec SniperMaterializationRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal sniper materialization record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append sniper materialization record: %w", err)
	}
	return nil
}

// ReadRecentSniperMaterializations reads records inside the [now-window, now] range.
// Malformed historical lines are skipped so one bad record does not disable the signal.
func ReadRecentSniperMaterializations(path string, now time.Time, window time.Duration) (records []SniperMaterializationRecord, err error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open sniper materialization history: %w", err)
	}
	defer closeSniperMaterializationFile(f, &err)

	return scanSniperMaterializations(f, now, window)
}

// closeSniperMaterializationFile closes f, surfacing the close error via *err only
// when the caller doesn't already have an error to report (a real read/write/parse
// failure takes priority over a close failure).
func closeSniperMaterializationFile(f *os.File, err *error) {
	if closeErr := f.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("close sniper materialization history: %w", closeErr)
	}
}

func scanSniperMaterializations(r io.Reader, now time.Time, window time.Duration) ([]SniperMaterializationRecord, error) {
	cutoff := now.Add(-window)
	var records []SniperMaterializationRecord
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if rec, ok := parseSniperMaterializationLine(scanner.Bytes(), cutoff, now); ok {
			records = append(records, rec)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan sniper materialization history: %w", err)
	}
	return records, nil
}

// parseSniperMaterializationLine parses one JSONL line and reports whether it falls
// inside [cutoff, now) with a non-empty target path. Malformed lines report ok=false
// rather than an error — one bad historical record must not disable the signal.
func parseSniperMaterializationLine(line []byte, cutoff, now time.Time) (rec SniperMaterializationRecord, ok bool) {
	if err := json.Unmarshal(line, &rec); err != nil {
		return rec, false
	}
	if rec.TargetPath == "" || rec.MaterializedAt.Before(cutoff) || rec.MaterializedAt.After(now) {
		return rec, false
	}
	return rec, true
}

// SniperConflictSignals builds threshold-qualified F3 conflict signals from current conflicts and recent records.
func SniperConflictSignals(basePath string, conflictedPaths []string, records []SniperMaterializationRecord) []SniperConflictSignal {
	recentByTarget := make(map[string]SniperMaterializationRecord, len(records))
	recentTargets := make([]string, 0, len(records))
	for _, rec := range records {
		if _, seen := recentByTarget[rec.TargetPath]; seen {
			continue
		}
		recentByTarget[rec.TargetPath] = rec
		recentTargets = append(recentTargets, rec.TargetPath)
	}

	attributed := ClassifyConflictedTargets(conflictedPaths, recentTargets)
	if !F3ConflictThresholdMet(len(attributed)) {
		return nil
	}

	signals := make([]SniperConflictSignal, 0, len(attributed))
	for _, target := range attributed {
		rec := recentByTarget[target]
		signalBasePath := basePath
		if signalBasePath == "" {
			signalBasePath = rec.BasePath
		}
		signals = append(signals, SniperConflictSignal{
			MissionID:     rec.MissionID,
			BasePath:      signalBasePath,
			TargetPath:    target,
			ConflictCount: len(attributed),
		})
	}
	return signals
}

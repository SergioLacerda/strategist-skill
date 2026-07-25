package telemetry

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
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
func AppendSniperMaterialization(path string, rec SniperMaterializationRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sniper materialization history dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open sniper materialization history: %w", err)
	}
	line, err := json.Marshal(rec)
	if err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("close sniper materialization history after marshal failure: %w", closeErr)
		}
		return fmt.Errorf("marshal sniper materialization record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("close sniper materialization history after append failure: %w", closeErr)
		}
		return fmt.Errorf("append sniper materialization record: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close sniper materialization history: %w", err)
	}
	return nil
}

// ReadRecentSniperMaterializations reads records inside the [now-window, now] range.
// Malformed historical lines are skipped so one bad record does not disable the signal.
func ReadRecentSniperMaterializations(path string, now time.Time, window time.Duration) ([]SniperMaterializationRecord, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open sniper materialization history: %w", err)
	}

	cutoff := now.Add(-window)
	var records []SniperMaterializationRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var rec SniperMaterializationRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.TargetPath == "" || rec.MaterializedAt.Before(cutoff) || rec.MaterializedAt.After(now) {
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return nil, fmt.Errorf("close sniper materialization history after scan failure: %w", closeErr)
		}
		return nil, fmt.Errorf("scan sniper materialization history: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close sniper materialization history: %w", err)
	}
	return records, nil
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

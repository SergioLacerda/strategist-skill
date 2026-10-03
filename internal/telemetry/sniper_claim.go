package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// SniperClaimHistoryRelPath is relative to the .strategist runtime root.
	SniperClaimHistoryRelPath = "memory/sniper-claims.jsonl"
	// SniperClaimWindow is the rolling window a caller reads claim history
	// over before checking for collisions — a session claiming a target more
	// than this long ago is treated as having released it (it either
	// committed or the mission ended), mirroring
	// SniperMaterializationWindow's own "caller-tracked rolling window"
	// framing for the sibling Git-conflict signal.
	SniperClaimWindow = 30 * 24 * time.Hour
)

// SniperClaimRecord records one mission's claim of a target path — appended
// when Sniper (or the parent-agent-embodied native role standing in for it) begins
// materializing a documentation target, before either commit or
// mission close. This is the claim-collision half of ADR-0008's F3 revisit
// tripwire (docs/adr/0008-single-session-assumption.md § F3 revisit
// tripwire): two or more distinct Sniper sessions claiming the same target
// before either commits, the signal sniper_conflict.go's
// f3ConflictThreshold doc comment names as "not instrumented here."
type SniperClaimRecord struct {
	// ExecutionEntryID correlates an idempotent execution-entry projection.
	// Empty is retained for historical JSONL compatibility.
	ExecutionEntryID string    `json:"execution_entry_id,omitempty"`
	MissionID        string    `json:"mission_id"`
	BasePath         string    `json:"base_path"`
	TargetPath       string    `json:"target_path"`
	PackageDigest    string    `json:"package_digest,omitempty"`
	ClaimedAt        time.Time `json:"claimed_at"`
}

// SniperClaimHistoryPath returns the default runtime memory path for Sniper claim history.
func SniperClaimHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(SniperClaimHistoryRelPath))
}

// AppendSniperClaim appends rec as one JSONL record, mirroring
// AppendSniperMaterialization's own append-only jsonl convention.
func AppendSniperClaim(path string, rec SniperClaimRecord) (err error) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return fmt.Errorf("create sniper claim history dir: %w", mkdirErr)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // G304: claim history path is owned by runtime memory
	if err != nil {
		return fmt.Errorf("open sniper claim history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close sniper claim history")
	if rec.ExecutionEntryID != "" {
		if exists, readErr := sniperClaimProjectionExists(path, rec); readErr != nil {
			return readErr
		} else if exists {
			return nil
		}
	}

	return writeSniperClaimLine(f, rec)
}

func sniperClaimProjectionExists(path string, want SniperClaimRecord) (bool, error) {
	f, err := os.Open(path) //nolint:gosec // runtime-owned history
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open sniper claim history: %w", err)
	}
	defer f.Close() //nolint:errcheck
	s := newJSONLScanner(f)
	for s.Scan() {
		var got SniperClaimRecord
		if json.Unmarshal(s.Bytes(), &got) == nil && got.ExecutionEntryID == want.ExecutionEntryID && got.TargetPath == want.TargetPath {
			return true, nil
		}
	}
	if err := jsonlScannerErr(s, "scan sniper claim history"); err != nil {
		return false, err
	}
	return false, nil
}

func writeSniperClaimLine(f *os.File, rec SniperClaimRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal sniper claim record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append sniper claim record: %w", err)
	}
	return nil
}

// ReadRecentSniperClaims reads records inside the [now-window, now] range.
// Malformed historical lines are skipped so one bad record does not disable
// the signal — same contract as ReadRecentSniperMaterializations.
func ReadRecentSniperClaims(path string, now time.Time, window time.Duration) (records []SniperClaimRecord, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: claim history path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open sniper claim history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close sniper claim history")

	return scanSniperClaims(f, now, window)
}

func scanSniperClaims(r io.Reader, now time.Time, window time.Duration) ([]SniperClaimRecord, error) {
	cutoff := now.Add(-window)
	var records []SniperClaimRecord
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if rec, ok := parseSniperClaimLine(scanner.Bytes(), cutoff, now); ok {
			records = append(records, rec)
		}
	}
	return records, jsonlScannerErr(scanner, "scan sniper claim history")
}

// parseSniperClaimLine parses one JSONL line and reports whether it falls
// inside [cutoff, now) with a non-empty target path. Malformed lines report
// ok=false rather than an error — one bad historical record must not
// disable the signal.
func parseSniperClaimLine(line []byte, cutoff, now time.Time) (rec SniperClaimRecord, ok bool) {
	if err := json.Unmarshal(line, &rec); err != nil {
		return rec, false
	}
	if rec.TargetPath == "" || rec.ClaimedAt.Before(cutoff) || rec.ClaimedAt.After(now) {
		return rec, false
	}
	return rec, true
}

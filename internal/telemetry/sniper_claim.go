package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
// when Sniper (or the parent-agent-embodied native role standing in for it)
// begins materializing a documentation target, before either commit or
// mission close. This is the claim-collision half of ADR-0008's F3 revisit
// tripwire (docs/adr/0008-single-session-assumption.md § F3 revisit
// tripwire): two or more distinct Sniper sessions claiming the same target
// before either commits, the signal sniper_conflict.go's
// f3ConflictThreshold doc comment names as "not instrumented here."
type SniperClaimRecord struct {
	MissionID  string    `json:"mission_id"`
	BasePath   string    `json:"base_path"`
	TargetPath string    `json:"target_path"`
	ClaimedAt  time.Time `json:"claimed_at"`
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

	return writeSniperClaimLine(f, rec)
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

// ClaimCollisionSignal reports two or more distinct missions having claimed
// the same target path within the caller-supplied window — the
// claim-collision half of ADR-0008's F3 revisit tripwire.
type ClaimCollisionSignal struct {
	BasePath   string
	TargetPath string
	// MissionIDs is the sorted, deduplicated set of distinct missions that
	// claimed TargetPath within the window.
	MissionIDs []string
}

// DetectClaimCollisions groups records by TargetPath and reports one
// ClaimCollisionSignal per target claimed by two or more distinct
// MissionIDs. A single mission claiming (or re-claiming) the same target
// multiple times is not a collision — only distinct missions targeting the
// same path count, matching ADR-0008's own framing ("two or more distinct
// Sniper sessions claiming the same target").
func DetectClaimCollisions(records []SniperClaimRecord) []ClaimCollisionSignal {
	if len(records) == 0 {
		return nil
	}
	missionsByTarget := make(map[string]map[string]bool)
	basePathByTarget := make(map[string]string)
	var targetOrder []string
	for _, rec := range records {
		if _, ok := missionsByTarget[rec.TargetPath]; !ok {
			missionsByTarget[rec.TargetPath] = make(map[string]bool)
			basePathByTarget[rec.TargetPath] = rec.BasePath
			targetOrder = append(targetOrder, rec.TargetPath)
		}
		missionsByTarget[rec.TargetPath][rec.MissionID] = true
	}

	var signals []ClaimCollisionSignal
	for _, target := range targetOrder {
		missions := missionsByTarget[target]
		if !ClaimCollisionThresholdMet(len(missions)) {
			continue
		}
		ids := make([]string, 0, len(missions))
		for id := range missions {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		signals = append(signals, ClaimCollisionSignal{
			BasePath:   basePathByTarget[target],
			TargetPath: target,
			MissionIDs: ids,
		})
	}
	return signals
}

// ClaimCollisionThresholdMet reports whether distinctMissionCount meets
// ADR-0008's F3 revisit tripwire threshold for claim-collision attribution:
// two or more distinct missions claiming the same target.
func ClaimCollisionThresholdMet(distinctMissionCount int) bool {
	return distinctMissionCount >= 2
}

// FormatClaimCollisionSignal returns a canonical progress-contract line for a claim collision signal.
func FormatClaimCollisionSignal(s ClaimCollisionSignal) string {
	return fmt.Sprintf(
		"[Strategist] signal=sniper_claim_collision base_path=%s target=%s missions=%s",
		SanitizePath(s.BasePath), SanitizePath(s.TargetPath), strings.Join(s.MissionIDs, ","),
	)
}

// EmitClaimCollisionSignal logs the signal through slog with canonical attributes.
func EmitClaimCollisionSignal(s ClaimCollisionSignal) {
	slog.Info(
		FormatClaimCollisionSignal(s),
		AttrBasePath, SanitizePath(s.BasePath),
		AttrTarget, SanitizePath(s.TargetPath),
		AttrClaimMissionIDs, strings.Join(s.MissionIDs, ","),
	)
}

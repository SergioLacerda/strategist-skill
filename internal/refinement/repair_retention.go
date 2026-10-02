package refinement

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

type repairTombstone struct {
	SchemaVersion              string `json:"schema_version"`
	MissionID                  string `json:"mission_id"`
	Event                      string `json:"event"`
	OriginalPackageDigest      string `json:"original_package_digest"`
	OriginalApprovalGateDigest string `json:"original_approval_gate_digest,omitempty"`
	OriginalPhase              string `json:"original_phase"`
	OriginalState              string `json:"original_state"`
	ResultPhase                string `json:"result_phase"`
	ResultState                string `json:"result_state"`
	HandoffAttempt             int    `json:"handoff_attempt"`
	Reason                     string `json:"reason"`
	RecordedAt                 string `json:"recorded_at"`
}

// RetainRepairEvidence records the rejected package identity and both sides of
// the repair transition. It is a tombstone rather than a second canonical
// package, so later amendments can reconstruct what was corrected without
// authorizing the rejected bytes.
func RetainRepairEvidence(basePath string, before, after domain.MissionEngineStatus, packageDigest, reason string, now time.Time) error {
	if basePath == "" || before.MissionID == "" || before.MissionID != after.MissionID || packageDigest == "" || reason == "" {
		return fmt.Errorf("refinement: repair tombstone identity is incomplete")
	}
	dir := filepath.Join(basePath, "refined", before.MissionID, ".repair-evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("refinement: create repair evidence: %w", err)
	}
	number, err := nextRepairNumber(dir)
	if err != nil {
		return err
	}
	return writeRepairTombstone(dir, number, repairTombstone{
		SchemaVersion:              "strategist-repair-tombstone/v1",
		MissionID:                  before.MissionID,
		Event:                      "refinement_artifact_invalid",
		OriginalPackageDigest:      packageDigest,
		OriginalApprovalGateDigest: before.ApprovalGatePackageDigest,
		OriginalPhase:              string(before.Phase), OriginalState: string(before.State),
		ResultPhase: string(after.Phase), ResultState: string(after.State), HandoffAttempt: before.HandoffAttempt,
		Reason: reason, RecordedAt: now.UTC().Format(time.RFC3339Nano),
	})
}

func writeRepairTombstone(dir string, number int, marker repairTombstone) error {
	raw, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("refinement: encode repair evidence: %w", err)
	}
	target := filepath.Join(dir, fmt.Sprintf("repair-%03d.json", number))
	tmp, err := os.CreateTemp(dir, ".repair-*.tmp")
	if err != nil {
		return fmt.Errorf("refinement: create repair evidence: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() //nolint:errcheck // best-effort cleanup after atomic publication
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close() //nolint:errcheck // preserve the write error
		return fmt.Errorf("refinement: write repair evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("refinement: close repair evidence: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("refinement: publish repair evidence: %w", err)
	}
	return nil
}

func nextRepairNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("refinement: read repair evidence: %w", err)
	}
	highest := 0
	for _, entry := range entries {
		number, convErr := repairNumber(entry.Name())
		if convErr != nil {
			return 0, convErr
		}
		if convErr == nil && number > highest {
			highest = number
		}
	}
	return highest + 1, nil
}

func repairNumber(name string) (int, error) {
	if !strings.HasPrefix(name, "repair-") {
		return 0, nil
	}
	number, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "repair-"), ".json"))
	if err != nil {
		return 0, fmt.Errorf("refinement: malformed repair evidence %q", name)
	}
	return number, nil
}

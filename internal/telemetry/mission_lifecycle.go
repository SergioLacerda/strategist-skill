package telemetry

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionLifecycleRelPath is the runtime-root-relative path of the append-only
// mission lifecycle ledger: chronological mission and phase markers derived
// from engine transitions. It is separate from mission state (ADR-0069).
const MissionLifecycleRelPath = "memory/mission-lifecycle.jsonl"

// MissionLifecycleSchemaVersion is the only record schema version.
const MissionLifecycleSchemaVersion = "1"

// MissionLifecycleRecord is one JSONL line of the lifecycle ledger. Timestamps
// are milestones; the ledger holds no token data.
type MissionLifecycleRecord struct {
	SchemaVersion string `json:"schema_version"`
	MarkerID      string `json:"marker_id"`
	MarkerKind    string `json:"marker_kind"`
	MissionID     string `json:"mission_id"`
	Phase         string `json:"phase"`
	PhaseRunID    string `json:"phase_run_id,omitempty"`
	EndReason     string `json:"end_reason,omitempty"`
	Stage         string `json:"stage,omitempty"`
	ObservedAt    string `json:"observed_at"`
}

// MissionLifecyclePath returns the default ledger path under strategistRoot.
func MissionLifecyclePath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(MissionLifecycleRelPath))
}

var lifecycleMarkerKinds = map[string]bool{
	string(domain.MarkerMissionStarted): true, string(domain.MarkerPhaseStarted): true,
	string(domain.MarkerPhaseEnded): true, string(domain.MarkerMissionEnded): true,
}

var lifecyclePhases = map[string]bool{
	string(domain.PhaseBootstrap): true, string(domain.PhaseIntake): true, string(domain.PhaseDiscovery): true,
	string(domain.PhaseRefinement): true, string(domain.PhaseApprovalGate): true,
	string(domain.PhaseExecution): true, string(domain.PhaseDone): true,
}

var lifecycleEndReasons = map[string]bool{
	string(domain.EndCompleted): true, string(domain.EndRevisionRequested): true, string(domain.EndBlocked): true,
	string(domain.EndRejected): true, string(domain.EndTimedOut): true,
	string(domain.EndAnalysisDelivered): true, string(domain.EndADRClosed): true,
}

// ValidateMissionLifecycleRecord checks a record before it is written.
func ValidateMissionLifecycleRecord(rec MissionLifecycleRecord) error {
	return errors.Join(lifecycleIdentityErrors(rec), lifecyclePhaseErrors(rec), lifecycleEndErrors(rec))
}

func lifecycleIdentityErrors(rec MissionLifecycleRecord) error {
	var errs []error
	if rec.SchemaVersion != MissionLifecycleSchemaVersion {
		errs = append(errs, fmt.Errorf("schema_version must be %q, got %q", MissionLifecycleSchemaVersion, rec.SchemaVersion))
	}
	if rec.MarkerID == "" {
		errs = append(errs, errors.New("marker_id is required"))
	}
	if rec.MissionID == "" {
		errs = append(errs, errors.New("mission_id is required"))
	}
	if _, err := time.Parse(time.RFC3339, rec.ObservedAt); err != nil {
		errs = append(errs, fmt.Errorf("observed_at must be RFC 3339: %w", err))
	}
	return errors.Join(errs...)
}

func lifecyclePhaseErrors(rec MissionLifecycleRecord) error {
	var errs []error
	if !lifecycleMarkerKinds[rec.MarkerKind] {
		errs = append(errs, fmt.Errorf("marker_kind %q is not a lifecycle marker", rec.MarkerKind))
	}
	if !lifecyclePhases[rec.Phase] {
		errs = append(errs, fmt.Errorf("phase %q is not a canonical phase", rec.Phase))
	}
	phaseKind := rec.MarkerKind == string(domain.MarkerPhaseStarted) || rec.MarkerKind == string(domain.MarkerPhaseEnded)
	if phaseKind && rec.PhaseRunID == "" {
		errs = append(errs, errors.New("phase_run_id is required on phase markers"))
	}
	return errors.Join(errs...)
}

func lifecycleEndErrors(rec MissionLifecycleRecord) error {
	isEnd := rec.MarkerKind == string(domain.MarkerPhaseEnded) || rec.MarkerKind == string(domain.MarkerMissionEnded)
	switch {
	case isEnd && !lifecycleEndReasons[rec.EndReason]:
		return fmt.Errorf("end_reason %q is not a lifecycle end reason", rec.EndReason)
	case !isEnd && rec.EndReason != "":
		return errors.New("end_reason is only valid on end markers")
	}
	return nil
}

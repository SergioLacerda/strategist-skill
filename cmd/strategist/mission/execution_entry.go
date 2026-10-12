package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
)

// executionEntry is a durable intent record. Filesystem operations cannot be
// atomic together, so replaying this record is the transaction boundary.
type executionEntry struct {
	ID                string                     `json:"id"`
	MissionID         string                     `json:"mission_id"`
	PackageDigest     string                     `json:"package_digest"`
	Targets           []string                   `json:"targets"`
	TargetMetadata    []domain.ExecutionTarget   `json:"target_metadata,omitempty"`
	ExecutionSnapshot *domain.ExecutionSnapshot  `json:"execution_snapshot,omitempty"`
	Outcome           handoff.Outcome            `json:"outcome"`
	DesiredStatus     domain.MissionEngineStatus `json:"desired_status"`
	StateSaved        bool                       `json:"state_saved"`
	SideQuestSealed   bool                       `json:"side_quest_sealed"`
	OutcomeConsumed   bool                       `json:"outcome_consumed"`
	ClaimsRecorded    bool                       `json:"claims_recorded"`
	Completed         bool                       `json:"completed"`
}

// I/O seams keep failure-path tests hermetic while production uses os calls.
var executionEntryLink = os.Link
var executionEntryRename = os.Rename
var executionEntryConsume = livemission.ConsumeHandoffOutcome
var executionEntryRecordClaims = livemission.RecordSniperClaimsForEntry
var executionEntryRecordSnapshotClaims = livemission.RecordSniperClaimsForSnapshot

func executionEntryPath(root, missionID string) string {
	return filepath.Join(root, "missions", "execution-entry", missionID+".json")
}

func prepareExecutionEntry(root, missionID, digest string, targets []string, targetMetadata []domain.ExecutionTarget, outcome *handoff.Outcome, desired domain.MissionEngineStatus) (*executionEntry, error) {
	metadata := targetMetadata
	if len(metadata) == 0 && len(targets) > 0 {
		metadata = executionTargetsFromPaths(targets)
	}
	snapshot, err := domain.SnapshotFromTargets(missionID, metadata)
	if err != nil {
		return nil, fmt.Errorf("prepare execution entry: build execution snapshot: %w", err)
	}
	return prepareExecutionEntryWithSnapshot(root, missionID, digest, targets, targetMetadata, snapshot, outcome, desired)
}

func prepareExecutionEntryWithSnapshot(root, missionID, digest string, targets []string, targetMetadata []domain.ExecutionTarget, snapshot domain.ExecutionSnapshot, outcome *handoff.Outcome, desired domain.MissionEngineStatus) (*executionEntry, error) {
	if outcome == nil {
		return nil, nil
	}
	path := executionEntryPath(root, missionID)
	existing, err := loadExecutionEntry(path)
	if err == nil {
		return reconcileExecutionEntryWithSnapshot(root, existing, missionID, digest, targets, targetMetadata, snapshot, outcome, desired)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	e := newExecutionEntryWithSnapshot(missionID, digest, targets, targetMetadata, &snapshot, outcome, desired)
	if err := saveNewExecutionEntry(path, e); err != nil {
		return nil, err
	}
	return &e, nil
}

func newExecutionEntryWithSnapshot(missionID, digest string, targets []string, targetMetadata []domain.ExecutionTarget, snapshot *domain.ExecutionSnapshot, outcome *handoff.Outcome, desired domain.MissionEngineStatus) executionEntry {
	return executionEntry{
		ID: fmt.Sprintf("%s-%d", missionID, time.Now().UTC().UnixNano()), MissionID: missionID,
		PackageDigest: digest, Targets: append([]string(nil), targets...), TargetMetadata: append([]domain.ExecutionTarget(nil), targetMetadata...), ExecutionSnapshot: snapshot, Outcome: *outcome, DesiredStatus: desired,
	}
}

func reconcileExecutionEntryWithSnapshot(root string, existing executionEntry, missionID, digest string, targets []string, targetMetadata []domain.ExecutionTarget, snapshot domain.ExecutionSnapshot, outcome *handoff.Outcome, desired domain.MissionEngineStatus) (*executionEntry, error) {
	if executionEntryNeedsReplacement(existing, digest, outcome) {
		e := newExecutionEntryWithSnapshot(missionID, digest, targets, targetMetadata, &snapshot, outcome, desired)
		if err := saveExecutionEntry(root, &e); err != nil {
			return nil, err
		}
		return &e, nil
	}
	if !executionEntryMatches(existing, missionID, digest, targetMetadata, snapshot, outcome, desired) {
		return nil, fmt.Errorf("execution entry tampered: identity mismatch")
	}
	return &existing, nil
}

func executionEntryNeedsReplacement(existing executionEntry, digest string, outcome *handoff.Outcome) bool {
	return existing.Completed && (existing.PackageDigest != digest || existing.Outcome.Integrity != outcome.Integrity)
}

func executionEntryMatches(existing executionEntry, missionID, digest string, targetMetadata []domain.ExecutionTarget, snapshot domain.ExecutionSnapshot, outcome *handoff.Outcome, desired domain.MissionEngineStatus) bool {
	if existing.MissionID != missionID || existing.PackageDigest != digest || existing.Outcome.Integrity != outcome.Integrity {
		return false
	}
	if !reflect.DeepEqual(existing.DesiredStatus, desired) || !reflect.DeepEqual(existing.TargetMetadata, targetMetadata) {
		return false
	}
	return existing.ExecutionSnapshot == nil || reflect.DeepEqual(*existing.ExecutionSnapshot, snapshot)
}

func loadExecutionEntry(path string) (executionEntry, error) {
	var e executionEntry
	raw, err := os.ReadFile(path) //nolint:gosec // path is the validated mission execution-entry path
	if err != nil {
		return e, fmt.Errorf("read execution entry %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return e, fmt.Errorf("execution entry tampered: %w", err)
	}
	return e, nil
}

func recoverExecutionOutcome(root string, e *executionEntry) error {
	consumed, err := handoff.NewOutcomeStore(root).Consumed(e.Outcome)
	if err != nil {
		return fmt.Errorf("read execution outcome: %w", err)
	}
	if !consumed {
		if err := executionEntryConsume(root, e.Outcome); err != nil {
			return fmt.Errorf("recover outcome consumption: %w", err)
		}
	}
	e.OutcomeConsumed = true
	return nil
}

func recoverExecutionClaims(root, basePath, missionID string, e *executionEntry) error {
	var err error
	if e.ExecutionSnapshot != nil && len(e.ExecutionSnapshot.Groups) > 0 {
		_, err = executionEntryRecordSnapshotClaims(root, basePath, missionID, e.ID, *e.ExecutionSnapshot, time.Now().UTC())
	} else {
		_, err = executionEntryRecordClaims(root, basePath, missionID, e.ID, executionTargetsForEntry(e), time.Now().UTC())
	}
	if err != nil {
		return fmt.Errorf("recover sniper claims: %w", err)
	}
	e.ClaimsRecorded, e.Completed = true, true
	if err := saveExecutionEntry(root, e); err != nil {
		return fmt.Errorf("finalize execution entry: %w", err)
	}
	return nil
}

// recoverExecutionEntry resumes the durable projections left by a returned
// I/O error. Outcome and claim writes are idempotent by their recorded entry.
func recoverExecutionEntry(root, basePath, missionID string, persisted domain.MissionEngineStatus) error {
	var entry executionEntry
	err := application.RecoverMissionExecution(application.RecoverExecutionRequest{
		Root: root, BasePath: basePath, MissionID: missionID, PersistedStatus: persisted,
	}, application.RecoverExecutionPorts{
		Load: func(root, missionID string) (application.ExecutionRecoverySnapshot, bool, error) {
			loaded, recoverable, err := loadRecoverableExecutionEntry(root, missionID)
			if err != nil || !recoverable {
				return application.ExecutionRecoverySnapshot{}, recoverable, err
			}
			entry = loaded
			return executionRecoverySnapshot(loaded), true, nil
		},
		MarkStateSaved: func(root string, _ application.ExecutionRecoverySnapshot) error {
			entry.StateSaved = true
			return saveExecutionEntry(root, &entry)
		},
		RecoverOutcome: func(root string, _ handoff.Outcome) error {
			return recoverExecutionOutcome(root, &entry)
		},
		CompleteClaims: func(root, basePath, missionID string, _ application.ExecutionRecoverySnapshot) error {
			return recoverExecutionClaims(root, basePath, missionID, &entry)
		},
	})
	if err != nil {
		return fmt.Errorf("recover mission execution: %w", err)
	}
	return nil
}

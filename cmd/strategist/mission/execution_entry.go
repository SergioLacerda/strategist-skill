package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
)

// executionEntry is a durable intent record. Filesystem operations cannot be
// atomic together, so replaying this record is the transaction boundary.
type executionEntry struct {
	ID              string                     `json:"id"`
	MissionID       string                     `json:"mission_id"`
	PackageDigest   string                     `json:"package_digest"`
	Targets         []string                   `json:"targets"`
	Outcome         handoff.Outcome            `json:"outcome"`
	DesiredStatus   domain.MissionEngineStatus `json:"desired_status"`
	StateSaved      bool                       `json:"state_saved"`
	SideQuestSealed bool                       `json:"side_quest_sealed"`
	OutcomeConsumed bool                       `json:"outcome_consumed"`
	ClaimsRecorded  bool                       `json:"claims_recorded"`
	Completed       bool                       `json:"completed"`
}

// I/O seams keep failure-path tests hermetic while production uses os calls.
var executionEntryLink = os.Link
var executionEntryRename = os.Rename
var executionEntryConsume = livemission.ConsumeHandoffOutcome
var executionEntryRecordClaims = livemission.RecordSniperClaimsForEntry

func executionEntryPath(root, missionID string) string {
	return filepath.Join(root, "missions", "execution-entry", missionID+".json")
}

func prepareExecutionEntry(root, missionID, digest string, targets []string, outcome *handoff.Outcome, desired domain.MissionEngineStatus) (*executionEntry, error) {
	if outcome == nil {
		return nil, nil
	}
	path := executionEntryPath(root, missionID)
	existing, err := loadExecutionEntry(path)
	if err == nil {
		return reconcileExecutionEntry(root, existing, missionID, digest, targets, outcome, desired)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	e := newExecutionEntry(missionID, digest, targets, outcome, desired)
	if err := saveNewExecutionEntry(path, e); err != nil {
		return nil, err
	}
	return &e, nil
}

func newExecutionEntry(missionID, digest string, targets []string, outcome *handoff.Outcome, desired domain.MissionEngineStatus) executionEntry {
	return executionEntry{
		ID: fmt.Sprintf("%s-%d", missionID, time.Now().UTC().UnixNano()), MissionID: missionID,
		PackageDigest: digest, Targets: append([]string(nil), targets...), Outcome: *outcome, DesiredStatus: desired,
	}
}

func reconcileExecutionEntry(root string, existing executionEntry, missionID, digest string, targets []string, outcome *handoff.Outcome, desired domain.MissionEngineStatus) (*executionEntry, error) {
	if existing.Completed && (existing.PackageDigest != digest || existing.Outcome.Integrity != outcome.Integrity) {
		e := newExecutionEntry(missionID, digest, targets, outcome, desired)
		if err := saveExecutionEntry(root, &e); err != nil {
			return nil, err
		}
		return &e, nil
	}
	if existing.MissionID != missionID || existing.PackageDigest != digest || existing.Outcome.Integrity != outcome.Integrity || !reflect.DeepEqual(existing.DesiredStatus, desired) {
		return nil, fmt.Errorf("execution entry tampered: identity mismatch")
	}
	return &existing, nil
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

func recoverExecutionState(root string, e *executionEntry, persisted domain.MissionEngineStatus) (bool, error) {
	if e.StateSaved {
		return true, nil
	}
	if !reflect.DeepEqual(e.DesiredStatus, persisted) {
		return false, nil
	}
	e.StateSaved = true
	if err := saveExecutionEntry(root, e); err != nil {
		return false, fmt.Errorf("mark recovered state: %w", err)
	}
	return true, nil
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
	if _, err := executionEntryRecordClaims(root, basePath, missionID, e.ID, e.Targets, time.Now().UTC()); err != nil {
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
	e, recoverable, err := loadRecoverableExecutionEntry(root, missionID)
	if err != nil {
		return err
	}
	if !recoverable {
		return nil
	}
	// A failed FSM save leaves an intent but no execution transition. It must
	// not consume authorization; the retried submit will reuse this entry.
	ready, err := recoverExecutionState(root, &e, persisted)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	if err := recoverExecutionOutcome(root, &e); err != nil {
		return err
	}
	return recoverExecutionClaims(root, basePath, missionID, &e)
}

func loadRecoverableExecutionEntry(root, missionID string) (executionEntry, bool, error) {
	e, err := loadExecutionEntry(executionEntryPath(root, missionID))
	if errors.Is(err, os.ErrNotExist) || e.Completed {
		return e, false, nil
	}
	if err != nil {
		return e, false, fmt.Errorf("load execution entry: %w", err)
	}
	if e.MissionID != missionID || e.ID == "" {
		return e, false, fmt.Errorf("execution entry tampered: identity mismatch")
	}
	return e, true, nil
}

package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// ADRTargetRequest asks for the reserved ADR path of an accepted OA-ADR.
type ADRTargetRequest struct {
	StrategistRoot string
	BasePath       string
	MissionID      string
	// Slug is the optional title slug used when the canonical directory
	// follows a numbered convention.
	Slug  string
	State domain.MissionState
	Now   time.Time
}

// ReserveADRTarget resolves the ADR destination at materialization time,
// reserves it once inside the accepted record, records the Sniper claim for it
// and returns the workspace-relative path. Repeated calls return the same
// path; a reservation interrupted before its claim is completed on replay.
// The caller must hold the mission lock.
func ReserveADRTarget(req ADRTargetRequest) (string, error) {
	rec, err := acceptedRecord(req)
	if err != nil {
		return "", err
	}
	rec, err = ensureReservedAndClaimed(req, rec)
	return rec.ReservedPath, err
}

func acceptedRecord(req ADRTargetRequest) (AcceptedSideQuest, error) {
	if req.State != domain.StateExecution {
		return AcceptedSideQuest{}, fmt.Errorf("adr_target_wrong_state: the ADR target is resolved at materialization, mission is %s", req.State)
	}
	rec, found, err := LoadAcceptedSideQuest(req.StrategistRoot, req.MissionID)
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	if !found || rec.Decision != SideQuestAccepted {
		return AcceptedSideQuest{}, fmt.Errorf("adr_side_quest_not_accepted: %s has no accepted record", OAADRSideQuestID(req.MissionID))
	}
	return rec, nil
}

func ensureReservedAndClaimed(req ADRTargetRequest, rec AcceptedSideQuest) (AcceptedSideQuest, error) {
	var err error
	if rec.ReservedPath == "" {
		if rec, err = reservePath(req, rec); err != nil {
			return AcceptedSideQuest{}, err
		}
	}
	if !rec.Claimed {
		err = claimReservedPath(req, rec)
	}
	return rec, err
}

func reservePath(req ADRTargetRequest, rec AcceptedSideQuest) (AcceptedSideQuest, error) {
	workspace := filepath.Dir(req.StrategistRoot)
	rel, err := destinationPath(req, rec, workspace)
	if err != nil {
		return AcceptedSideQuest{}, err
	}
	if _, statErr := os.Stat(absoluteTarget(workspace, rel)); statErr == nil {
		return AcceptedSideQuest{}, fmt.Errorf("adr_target_exists: %s already exists; it is never overwritten", rel)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return AcceptedSideQuest{}, fmt.Errorf("adr_target_unreadable: %w", statErr)
	}
	rec.ReservedPath = rel
	if !rec.Sealed { // execution entry implies gate_approved; heal an interrupted seal
		rec.Sealed, rec.SealedAt = true, nowStamp(req.Now)
	}
	return saveAcceptedSideQuest(req.StrategistRoot, rec)
}

func destinationPath(req ADRTargetRequest, rec AcceptedSideQuest, workspace string) (string, error) {
	if rec.Destination.CanonicalPath == "" {
		return workspaceRelative(workspace, filepath.Join(req.BasePath, rec.Destination.Fallback)), nil
	}
	dir := rec.Destination.CanonicalPath
	name, err := nextADRFilename(filepath.Join(workspace, dir), reservedNamesIn(req.StrategistRoot, req.MissionID, dir), req.MissionID, req.Slug)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(dir, name)), nil
}

func claimReservedPath(req ADRTargetRequest, rec AcceptedSideQuest) error {
	claim := telemetry.SniperClaimRecord{MissionID: req.MissionID, BasePath: req.BasePath, TargetPath: rec.ReservedPath, ClaimedAt: req.Now.UTC()}
	if err := telemetry.AppendSniperClaim(telemetry.SniperClaimHistoryPath(req.StrategistRoot), claim); err != nil {
		return fmt.Errorf("adr_target_claim_failed: %w", err)
	}
	rec.Claimed = true
	_, err := saveAcceptedSideQuest(req.StrategistRoot, rec)
	return err
}

func absoluteTarget(workspace, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(workspace, rel)
}

// workspaceRelative reports path relative to the workspace, or unchanged when
// it lies outside it.
func workspaceRelative(workspace, path string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil || rel == ".." || len(rel) > 2 && rel[:3] == ".."+string(filepath.Separator) {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

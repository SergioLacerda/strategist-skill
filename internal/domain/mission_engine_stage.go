package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CriticalHitStageApprovalDigest identifies the exact gated SHORT resolution
// accepted by the user. Critical Hit has no refined package to digest, so its
// execution request is bound to the persisted mission/stage correlation.
func CriticalHitStageApprovalDigest(status MissionEngineStatus) string {
	if status.Stage != StageShort || status.StageFeat != "critical_hit" || status.StageCorrelationID == "" {
		return ""
	}
	material := "critical-hit-stage/v1:" + status.MissionID + ":" + status.StageCorrelationID + ":" + status.StageFeat
	digest := sha256.Sum256([]byte(material))
	return "sha256:" + hex.EncodeToString(digest[:])
}

// RecordStageResolution binds the Role-owned Stage resolution to the mission
// before the activation event is submitted. It never advances the mission.
func (e *MissionEngine) RecordStageResolution(resolution StageResolution) (MissionEngineStatus, error) {
	if e == nil {
		return MissionEngineStatus{}, fmt.Errorf("mission engine: engine is nil")
	}
	if err := resolution.Validate(); err != nil {
		return e.status, fmt.Errorf("mission engine: stage resolution: %w", err)
	}
	if resolution.MissionID != "" && resolution.MissionID != e.status.MissionID {
		return e.status, fmt.Errorf("mission engine: stage resolution mission id %q does not match %q", resolution.MissionID, e.status.MissionID)
	}
	if resolution.Stage != StageShort || resolution.Feat != "critical_hit" || !resolution.GateRequired {
		return e.status, fmt.Errorf("mission engine: only a gated Critical Hit SHORT resolution can activate the short path")
	}
	if !earlyMissionPhase(e.status.Phase) {
		return e.status, fmt.Errorf("mission engine: stage resolution requires an early pipeline phase, got %q", e.status.Phase)
	}
	e.status.Stage = resolution.Stage
	e.status.StageFeat = resolution.Feat
	e.status.StageCorrelationID = resolution.CorrelationKey
	e.status.StageGateRequired = resolution.GateRequired
	e.status.StageGateApproved = false
	return e.status, nil
}

func (e *MissionEngine) submitCriticalHitIntent() (MissionEngineStatus, error) {
	if e.status.Phase != PhaseIntake && e.status.Phase != PhaseDiscovery {
		return e.status, ErrOutOfOrderPhaseSubmit{Current: e.status.Phase, Event: PhaseEvent(MissionEventCriticalHitIntent)}
	}
	if e.status.Stage != StageShort || e.status.StageFeat != "critical_hit" || !e.status.StageGateRequired {
		return e.status, fmt.Errorf("mission engine: Critical Hit intent requires a recorded gated SHORT resolution")
	}
	e.status.State = StateApprovalGate
	e.status.Phase = PhaseApprovalGate
	return e.status, nil
}

func (e *MissionEngine) validateCriticalHitGateEvent(event MissionEngineEvent) error {
	if !isCriticalHitGateEvent(event) {
		return e.rejectImplicitCriticalHitApproval()
	}
	return e.validateCriticalHitGateOutcome()
}

func isCriticalHitGateEvent(event MissionEngineEvent) bool {
	return event == MissionEventCriticalHitGateApproved || event == MissionEventCriticalHitGateDeclined
}

func (e *MissionEngine) rejectImplicitCriticalHitApproval() error {
	if e.status.Stage == StageShort && e.status.StageFeat == "critical_hit" && e.status.State == StateApprovalGate {
		return fmt.Errorf("mission engine: Critical Hit approval requires an explicit Critical Hit gate event")
	}
	return nil
}

func (e *MissionEngine) validateCriticalHitGateOutcome() error {
	if e.status.Stage != StageShort || e.status.StageFeat != "critical_hit" || e.status.State != StateApprovalGate || !e.status.StageGateRequired {
		return fmt.Errorf("mission engine: Critical Hit gate event requires an active gated SHORT approval state")
	}
	return nil
}

func (e *MissionEngine) recordCriticalHitGateOutcome(event MissionEngineEvent) {
	if event == MissionEventCriticalHitGateApproved {
		e.status.StageGateApproved = true
		e.status.ApprovalGatePackageDigest = CriticalHitStageApprovalDigest(e.status)
		return
	}
	if event == MissionEventCriticalHitGateDeclined {
		e.status.StageGateApproved = false
	}
}

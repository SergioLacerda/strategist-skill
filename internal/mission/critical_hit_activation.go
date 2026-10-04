package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const criticalHitRoute = "critical_hit"

// ActivateCriticalHitRoute turns the persisted Role-owned SHORT resolution
// into a mission-state transition. Route history and mission state are kept as
// separate evidence; a route record alone never makes execution eligible.
func ActivateCriticalHitRoute(
	root, missionID string,
	load func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error),
	save func(string, domain.MissionEngineStatus) error,
) (domain.MissionEngineStatus, error) {
	var status domain.MissionEngineStatus
	err := Lock(root, missionID, func() error {
		var err error
		status, err = activateCriticalHitRouteLocked(root, missionID, load, save)
		return err
	})
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	return status, nil
}

func activateCriticalHitRouteLocked(root, missionID string, load func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error), save func(string, domain.MissionEngineStatus) error) (domain.MissionEngineStatus, error) {
	engine, persisted, err := load(root, missionID)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("load mission for Critical Hit activation: %w", err)
	}
	idempotent, err := validateCriticalHitActivationStatus(persisted)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if idempotent {
		return persisted, nil
	}
	resolution, err := criticalHitResolution(root, missionID)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if _, err := engine.RecordStageResolution(resolution); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("record Critical Hit Stage resolution: %w", err)
	}
	status, err := engine.Submit(domain.MissionEventCriticalHitIntent)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("submit Critical Hit intent: %w", err)
	}
	if err := save(root, status); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("persist Critical Hit activation: %w", err)
	}
	return status, nil
}

func validateCriticalHitActivationStatus(status domain.MissionEngineStatus) (bool, error) {
	if status.Stage == domain.StageShort && status.StageFeat == criticalHitRoute {
		if map[domain.MissionState]bool{
			domain.StateApprovalGate: true,
			domain.StateExecution:    true,
			domain.StateDoneDelivery: true,
		}[status.State] {
			return true, nil
		}
		return false, fmt.Errorf("critical hit activation cannot replay from mission state %q", status.State)
	}
	if status.Phase != domain.PhaseIntake && status.Phase != domain.PhaseDiscovery {
		return false, fmt.Errorf("critical hit activation requires intake or discovery phase, got %q", status.Phase)
	}
	return false, nil
}

func criticalHitResolution(root, missionID string) (domain.StageResolution, error) {
	decision, err := persistedRouteDecision(root, missionID)
	if err != nil {
		return domain.StageResolution{}, fmt.Errorf("read Critical Hit route: %w", err)
	}
	if decision.SelectedRoute != criticalHitRoute {
		return domain.StageResolution{}, fmt.Errorf("critical hit activation requires route %q, got %q", criticalHitRoute, decision.SelectedRoute)
	}
	return domain.StageResolution{
		SchemaVersion:   domain.StageResolutionArtifactSchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Stage:           domain.Stage(decision.Stage),
		LegacyRoute:     decision.SelectedRoute,
		Role:            decision.StageRole,
		Feat:            decision.StageFeat,
		MissionID:       decision.MissionID,
		CorrelationKey:  decision.StageCorrelationID,
		GateRequired:    decision.StageGateRequired,
		PolicyVersion:   decision.StagePolicyVersion,
		Reason:          decision.StageReason,
	}, nil
}

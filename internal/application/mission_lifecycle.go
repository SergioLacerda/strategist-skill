package application

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionStartPorts are the persistence and policy boundaries needed to start
// a mission. The application service owns the ordering; adapters own the
// concrete filesystem, locking, and INITIATIVE implementations.
type MissionStartPorts struct {
	RequireNoExisting func(root, missionID string) error
	InitiativeStart   func(root, missionID string) error
	Save              func(root string, status domain.MissionEngineStatus) error
}

// StartMission creates and persists one mission after its preconditions and
// consultative INITIATIVE hook succeed. It never depends on Cobra or a
// concrete persistence implementation.
func StartMission(root, missionID string, ports MissionStartPorts) (domain.MissionEngineStatus, error) {
	if err := validateStartPorts(ports); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := ports.RequireNoExisting(root, missionID); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	engine, status, err := startEngine(missionID)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := consultInitiative(ports.InitiativeStart, root, missionID); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := ports.Save(root, engine.Status()); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission start: %w", err)
	}
	return status, nil
}

func validateStartPorts(ports MissionStartPorts) error {
	if ports.RequireNoExisting == nil || ports.Save == nil {
		return fmt.Errorf("mission start: persistence ports are required")
	}
	return nil
}

func startEngine(missionID string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: missionID})
	if err != nil {
		return nil, domain.MissionEngineStatus{}, fmt.Errorf("mission start: %w", err)
	}
	return engine, status, nil
}

func consultInitiative(consult func(root, missionID string) error, root, missionID string) error {
	if consult == nil {
		return nil
	}
	if err := consult(root, missionID); err != nil {
		return fmt.Errorf("mission start: initiative consultation: %w", err)
	}
	return nil
}

// LoadMissionStatus validates and projects a durable status through the
// application boundary without advancing the mission.
func LoadMissionStatus(load func(root, missionID string) (*domain.MissionEngine, domain.MissionEngineStatus, error), root, missionID string) (domain.MissionEngineStatus, error) {
	if load == nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission query: mission loader is required")
	}
	_, status, err := load(root, missionID)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if _, err := QueryMission(status); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	return status, nil
}

// RecordRouteRequest is the adapter-neutral route recording input.
type RecordRouteRequest struct {
	Root      string
	MissionID string
	Raw       []byte
}

// RecordRoute records a route through an injected runtime connector. The
// connector selects telemetry and persistence at the composition root.
func RecordRoute(ctx context.Context, request RecordRouteRequest, recorder func(context.Context, string, string, []byte) (bool, error)) (bool, error) {
	if recorder == nil {
		return false, fmt.Errorf("mission route: route recorder is required")
	}
	if request.Root == "" || request.MissionID == "" {
		return false, fmt.Errorf("mission route: root and mission id are required")
	}
	return recorder(ctx, request.Root, request.MissionID, append([]byte(nil), request.Raw...))
}

package application

import (
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// HandoffLifecyclePorts are the concrete mission-state boundaries needed by
// an Archivist handoff evaluation. The application service owns the
// lock/load/evaluate/save ordering; adapters provide filesystem and evaluator
// implementations.
type HandoffLifecyclePorts struct {
	Lock     func(root, missionID string, fn func() error) error
	Load     func(root, missionID string) (*domain.MissionEngine, domain.MissionEngineStatus, error)
	Save     func(root string, status domain.MissionEngineStatus) error
	Evaluate func(root, basePath string, engine *domain.MissionEngine) (domain.MissionEngineStatus, bool, error)
}

// RecordHandoffLifecycle executes one handoff evaluation inside the mission
// lock and persists state whenever the evaluator changed it, including when
// the evaluator also returns an error. This preserves crash-recovery and
// failed-evaluation semantics without coupling the application layer to a
// concrete handoff evaluator or telemetry package.
func RecordHandoffLifecycle(root, basePath, missionID string, ports HandoffLifecyclePorts) error {
	if err := validateHandoffLifecyclePorts(ports); err != nil {
		return err
	}
	return ports.Lock(root, missionID, func() error {
		return recordHandoffLocked(root, basePath, missionID, ports)
	})
}

func recordHandoffLocked(root, basePath, missionID string, ports HandoffLifecyclePorts) error {
	engine, _, err := ports.Load(root, missionID)
	if err != nil {
		return err
	}
	status, changed, evaluationErr := ports.Evaluate(root, basePath, engine)
	return persistHandoffState(root, status, changed, evaluationErr, ports.Save)
}

func persistHandoffState(root string, status domain.MissionEngineStatus, changed bool, evaluationErr error, save func(string, domain.MissionEngineStatus) error) error {
	if changed {
		if err := save(root, status); err != nil {
			return errors.Join(evaluationErr, fmt.Errorf("handoff_state_persist_failed: save mission state: %w", err))
		}
	}
	if evaluationErr != nil {
		return fmt.Errorf("record handoff: %w", evaluationErr)
	}
	return nil
}

func validateHandoffLifecyclePorts(ports HandoffLifecyclePorts) error {
	if ports.Lock == nil || ports.Load == nil || ports.Save == nil || ports.Evaluate == nil {
		return errors.New("handoff lifecycle ports are required")
	}
	return nil
}

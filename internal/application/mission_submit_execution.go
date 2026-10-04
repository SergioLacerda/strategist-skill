package application

import (
	"fmt"
	"reflect"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// SubmitExecutionCommitRequest identifies the post-FSM execution projections
// that must be committed in order.
type SubmitExecutionCommitRequest struct {
	Root         string
	BasePath     string
	MissionID    string
	Event        domain.MissionEngineEvent
	Outcome      *handoff.Outcome
	EntryPresent bool
}

// SubmitExecutionCommitPorts keep the durable outcome, execution journal, and
// Sniper claim implementations at the adapter boundary.
type SubmitExecutionCommitPorts struct {
	PersistOutcome     func(root string, outcome handoff.Outcome) error
	RecordEntryClaims  func(root, basePath, missionID string) error
	RecordSniperClaims func(root, basePath, missionID string, event domain.MissionEngineEvent) error
}

// CommitMissionExecution preserves the idempotent projection order used after
// a mission transition: consume outcome, then project claims.
func CommitMissionExecution(request SubmitExecutionCommitRequest, ports SubmitExecutionCommitPorts) error {
	if err := persistCommitOutcome(request, ports); err != nil {
		return err
	}
	claimed, err := recordEntryClaimsIfNeeded(request, ports)
	if err != nil {
		return err
	}
	if claimed {
		return nil
	}
	return recordFallbackClaims(request, ports)
}

func persistCommitOutcome(request SubmitExecutionCommitRequest, ports SubmitExecutionCommitPorts) error {
	if request.Outcome == nil {
		return nil
	}
	if ports.PersistOutcome == nil {
		return fmt.Errorf("mission submit: outcome persistence port is required")
	}
	return ports.PersistOutcome(request.Root, *request.Outcome)
}

func recordEntryClaimsIfNeeded(request SubmitExecutionCommitRequest, ports SubmitExecutionCommitPorts) (bool, error) {
	if !request.EntryPresent || request.Event != domain.MissionEventHandoffSatisfied {
		return false, nil
	}
	if ports.RecordEntryClaims == nil {
		return false, fmt.Errorf("mission submit: execution-entry claims port is required")
	}
	return true, ports.RecordEntryClaims(request.Root, request.BasePath, request.MissionID)
}

func recordFallbackClaims(request SubmitExecutionCommitRequest, ports SubmitExecutionCommitPorts) error {
	if ports.RecordSniperClaims == nil {
		return fmt.Errorf("mission submit: Sniper claims port is required")
	}
	return ports.RecordSniperClaims(request.Root, request.BasePath, request.MissionID, request.Event)
}

// ExecutionRecoverySnapshot is the application projection of the durable
// execution-entry state needed to decide whether replay may continue.
type ExecutionRecoverySnapshot struct {
	ID            string
	MissionID     string
	Targets       []string
	Outcome       handoff.Outcome
	DesiredStatus domain.MissionEngineStatus
	StateSaved    bool
	Completed     bool
}

// RecoverExecutionRequest identifies one mission replay attempt.
type RecoverExecutionRequest struct {
	Root            string
	BasePath        string
	MissionID       string
	PersistedStatus domain.MissionEngineStatus
}

// RecoverExecutionPorts expose the durable execution-entry operations without
// leaking its filesystem representation into the application package.
type RecoverExecutionPorts struct {
	Load           func(root, missionID string) (ExecutionRecoverySnapshot, bool, error)
	MarkStateSaved func(root string, snapshot ExecutionRecoverySnapshot) error
	RecoverOutcome func(root string, outcome handoff.Outcome) error
	CompleteClaims func(root, basePath, missionID string, snapshot ExecutionRecoverySnapshot) error
}

// RecoverMissionExecution replays a prepared execution entry in the same
// fail-safe order as the command path. A non-ready entry remains untouched.
func RecoverMissionExecution(request RecoverExecutionRequest, ports RecoverExecutionPorts) error {
	if err := validateRecoveryPorts(ports); err != nil {
		return err
	}
	snapshot, recoverable, err := ports.Load(request.Root, request.MissionID)
	if err != nil || !recoverable {
		return err
	}
	ready, err := recoverState(request, snapshot, ports.MarkStateSaved)
	if err != nil || !ready {
		return err
	}
	if err := ports.RecoverOutcome(request.Root, snapshot.Outcome); err != nil {
		return err
	}
	return ports.CompleteClaims(request.Root, request.BasePath, request.MissionID, snapshot)
}

func validateRecoveryPorts(ports RecoverExecutionPorts) error {
	if ports.Load == nil || ports.MarkStateSaved == nil || ports.RecoverOutcome == nil || ports.CompleteClaims == nil {
		return fmt.Errorf("mission submit: execution recovery ports are required")
	}
	return nil
}

func recoverState(request RecoverExecutionRequest, snapshot ExecutionRecoverySnapshot, mark func(string, ExecutionRecoverySnapshot) error) (bool, error) {
	if snapshot.StateSaved {
		return true, nil
	}
	if !reflect.DeepEqual(snapshot.DesiredStatus, request.PersistedStatus) {
		return false, nil
	}
	if err := mark(request.Root, snapshot); err != nil {
		return false, fmt.Errorf("mark recovered state: %w", err)
	}
	return true, nil
}

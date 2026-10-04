package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// SubmitMissionRequest is the application input for one authoritative FSM
// event. The gate digest is optional and is recorded atomically with the
// transition when present.
type SubmitMissionRequest struct {
	Event      domain.MissionEngineEvent
	GateDigest string
}

// SubmitMission applies the domain transition and records the approval-gate
// package digest as one application operation. Persistence and surrounding
// evidence guards remain at the composition root during migration.
func SubmitMission(engine *domain.MissionEngine, request SubmitMissionRequest) (domain.MissionEngineStatus, error) {
	if engine == nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: mission engine is required")
	}
	if request.Event == "" {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: event is required")
	}
	status, err := engine.Submit(request.Event)
	if err == nil && request.GateDigest != "" {
		status, err = engine.RecordApprovalGatePackageDigest(request.GateDigest)
	}
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	return status, nil
}

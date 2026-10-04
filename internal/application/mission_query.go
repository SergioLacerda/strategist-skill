package application

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// MissionSummary is the stable read-only DTO exposed to CLI consumers.
type MissionSummary struct {
	MissionID      string
	Phase          domain.PipelinePhase
	State          domain.MissionState
	HandoffAttempt int
}

// QueryMission projects durable mission status without advancing or
// authorizing the mission.
func QueryMission(status domain.MissionEngineStatus) (MissionSummary, error) {
	if status.MissionID == "" {
		return MissionSummary{}, fmt.Errorf("mission query: mission id is required")
	}
	return MissionSummary{
		MissionID: status.MissionID, Phase: status.Phase, State: status.State,
		HandoffAttempt: status.HandoffAttempt,
	}, nil
}

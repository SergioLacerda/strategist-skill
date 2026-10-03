package mission

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// recordedRouteDecision returns the selected route and canonical Stage Scout
// recorded for the mission, or an empty decision when none was recorded.
func recordedRouteDecision(strategistRoot, missionID string) (telemetry.RouteDecision, error) {
	decisions, err := telemetry.ReadRouteDecisions(telemetry.RouteDecisionHistoryPath(strategistRoot))
	if err != nil {
		return telemetry.RouteDecision{}, fmt.Errorf("read route decisions: %w", err)
	}
	for _, decision := range decisions {
		if decision.MissionID == missionID {
			return decision, nil
		}
	}
	return telemetry.RouteDecision{}, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

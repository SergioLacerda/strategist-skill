package mission

import "fmt"

// RequireRecordedRoute rejects intake completion for a mission that has no
// recorded Scout route decision: once Strategist is invoked, every mission
// passes through the initial route before discovery may start.
func RequireRecordedRoute(strategistRoot, missionID string) error {
	decision, err := recordedRouteDecision(strategistRoot, missionID)
	if err != nil {
		return err
	}
	if decision.MissionID == "" {
		return fmt.Errorf("error=route_decision_missing mission %q has no recorded route decision; record Scout's route_decision with `strategist mission route --mission-id %s` before submitting intake_done", missionID, missionID)
	}
	return nil
}

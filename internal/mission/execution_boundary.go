package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// executionEntryAction names the guarded transition in a bypass decision.
const executionEntryAction = "enter execution (handoff_challenge_passed)"

// RecordRouteDecision persists Scout's route_decision for a mission so the
// execution boundary can read it back. The decision must belong to missionID;
// an absent timestamp is filled in. It reports false when a decision for the
// mission was already recorded (the history is idempotent by mission_id).
func RecordRouteDecision(strategistRoot, missionID string, raw []byte) (bool, error) {
	var decision telemetry.RouteDecision
	if err := json.Unmarshal(raw, &decision); err != nil {
		return false, fmt.Errorf("route decision is not valid JSON: %w", err)
	}
	if decision.MissionID != missionID {
		return false, fmt.Errorf("route decision mission_id %q does not match mission %q", decision.MissionID, missionID)
	}
	if decision.Timestamp == "" {
		decision.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	line, err := json.Marshal(decision)
	if err != nil {
		return false, fmt.Errorf("encode route decision: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(telemetry.RouteDecisionHistoryPath(strategistRoot)), 0o755); err != nil { //nolint:gosec // G301: runtime memory directory
		return false, fmt.Errorf("create route decision directory: %w", err)
	}
	appended, err := telemetry.AppendRouteDecisionLine(telemetry.RouteDecisionHistoryPath(strategistRoot), string(line))
	if err != nil {
		return false, fmt.Errorf("append route decision: %w", err)
	}
	return appended, nil
}

// EvaluateExecutionEntry asks domain.EvaluatePipelineBypass whether the mission
// may enter execution. It is the live boundary the pure evaluation was missing:
// the route comes from Scout's recorded route_decision (strictest regime when
// none was recorded), the discovery, refinement and tasks evidence comes from
// the refined package under basePath, and the gate evidence comes from the
// mission engine itself, which reaches the handoff challenge only through an
// approved Approval Gate.
func EvaluateExecutionEntry(strategistRoot, basePath string, status domain.MissionEngineStatus) (domain.PipelineBypassDecision, error) {
	selected, err := recordedRoute(strategistRoot, status.MissionID)
	if err != nil {
		return domain.PipelineBypassDecision{}, err
	}
	refined := filepath.Join(basePath, "refined", status.MissionID)
	gateApproved := status.State == domain.StateHandoffChallenge
	return domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		Route:              domain.PipelineRouteForScoutRoute(selected),
		BasePath:           basePath,
		MissionID:          status.MissionID,
		AttemptedAction:    executionEntryAction,
		DiscoveryPresent:   fileExists(filepath.Join(refined, "analysis.md")),
		RefinementPresent:  fileExists(filepath.Join(refined, "proposal.md")) && fileExists(filepath.Join(refined, "design.md")),
		TasksPresent:       fileExists(filepath.Join(refined, "tasks.md")),
		GatePresented:      gateApproved,
		GateApproved:       gateApproved,
		DirectGateApproved: gateApproved,
	}), nil
}

// recordedRoute returns the selected_route Scout recorded for the mission, or ""
// when none was recorded.
func recordedRoute(strategistRoot, missionID string) (string, error) {
	decisions, err := telemetry.ReadRouteDecisions(telemetry.RouteDecisionHistoryPath(strategistRoot))
	if err != nil {
		return "", fmt.Errorf("read route decisions: %w", err)
	}
	for _, decision := range decisions {
		if decision.MissionID == missionID {
			return decision.SelectedRoute, nil
		}
	}
	return "", nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

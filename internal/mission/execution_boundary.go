package mission

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// executionEntryAction names the guarded transition in a bypass decision.
const executionEntryAction = "enter execution (handoff_challenge_satisfied)"

// RecordRouteDecision persists Scout's route_decision for a mission so the
// execution boundary can read it back. The decision must belong to missionID;
// an absent timestamp is filled in. It reports false when a decision for the
// mission was already recorded (the history is idempotent by mission_id).
func RecordRouteDecision(strategistRoot, missionID string, raw []byte) (bool, error) {
	decision, err := parseRouteDecision(missionID, raw)
	if err != nil {
		return false, err
	}
	appended, decision, err := appendRouteDecision(strategistRoot, decision)
	if err != nil {
		return false, err
	}
	if err := recordScoutRouteConfidence(strategistRoot, decision); err != nil {
		return appended, fmt.Errorf("record Scout route confidence: %w", err)
	}
	return appended, nil
}

// parseRouteDecision decodes Scout's decision, checks it belongs to missionID
// and fills an absent timestamp.
func parseRouteDecision(missionID string, raw []byte) (telemetry.RouteDecision, error) {
	var decision telemetry.RouteDecision
	if err := json.Unmarshal(raw, &decision); err != nil {
		return telemetry.RouteDecision{}, fmt.Errorf("route decision is not valid JSON: %w", err)
	}
	if decision.MissionID != missionID {
		return telemetry.RouteDecision{}, fmt.Errorf("route decision mission_id %q does not match mission %q", decision.MissionID, missionID)
	}
	if decision.Timestamp == "" {
		decision.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	return decision, nil
}

// appendRouteDecision appends the decision to the history. When one was already
// recorded for the mission it reports false and returns the persisted decision,
// which is the one the confidence claim must describe.
func appendRouteDecision(strategistRoot string, decision telemetry.RouteDecision) (bool, telemetry.RouteDecision, error) {
	line, err := json.Marshal(decision)
	if err != nil {
		return false, decision, fmt.Errorf("encode route decision: %w", err)
	}
	historyPath := telemetry.RouteDecisionHistoryPath(strategistRoot)
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil { //nolint:gosec // G301: runtime memory directory
		return false, decision, fmt.Errorf("create route decision directory: %w", err)
	}
	appended, err := telemetry.AppendRouteDecisionLine(historyPath, string(line))
	if err != nil {
		return false, decision, fmt.Errorf("append route decision: %w", err)
	}
	if appended {
		return true, decision, nil
	}
	persisted, err := persistedRouteDecision(strategistRoot, decision.MissionID)
	return false, persisted, err
}

func persistedRouteDecision(strategistRoot, missionID string) (telemetry.RouteDecision, error) {
	decisions, err := telemetry.ReadRouteDecisions(telemetry.RouteDecisionHistoryPath(strategistRoot))
	if err != nil {
		return telemetry.RouteDecision{}, fmt.Errorf("read persisted route decision: %w", err)
	}
	for _, decision := range decisions {
		if decision.MissionID == missionID {
			return decision, nil
		}
	}
	return telemetry.RouteDecision{}, fmt.Errorf("persisted route decision for mission %q was not found", missionID)
}

func recordScoutRouteConfidence(strategistRoot string, decision telemetry.RouteDecision) error {
	percent := int(math.Round(decision.RouteConfidence * 100))
	level, err := domain.ConfidenceLevelForPercent(percent)
	if err != nil {
		return fmt.Errorf("derive Scout route confidence level: %w", err)
	}
	kind := domain.ClaimKindAssertion
	statement := fmt.Sprintf("Scout selected route %q for request category %q.", decision.SelectedRoute, decision.RequestCategory)
	if level == domain.ConfidenceLow {
		kind = domain.ClaimKindQuestion
		statement = fmt.Sprintf("Is route %q appropriate for request category %q?", decision.SelectedRoute, decision.RequestCategory)
	}
	evidenceID := "scout-route-decision"
	evidence := []domain.Evidence{{
		ID: evidenceID, SourceRef: telemetry.RouteDecisionHistoryRelPath + "#mission_id=" + decision.MissionID,
		Class: domain.EvidenceClassExplicit, Confidence: level, ConfidencePercent: &percent,
	}}
	claim := domain.ConfidenceClaim{
		ID: "scout-route", Statement: statement, Agent: telemetry.ConfidenceAgentScout,
		CorrelationKey: "scout-route", ClaimKind: kind, ConfidencePercent: percent,
		ConfidenceLevel: level, EvidenceIDs: []string{evidenceID},
		EvidenceClasses: []string{domain.EvidenceClassExplicit}, CalibrationStatus: domain.CalibrationNoSample,
	}
	producer, err := telemetry.NewConfidenceProducerAdapter(
		telemetry.ConfidenceHistoryPath(strategistRoot), telemetry.ConfidenceAgentScout, decision.MissionID,
	)
	if err != nil {
		return fmt.Errorf("create Scout confidence producer: %w", err)
	}
	_, err = producer.RecordClaim(claim, evidence)
	if err != nil {
		return fmt.Errorf("persist Scout confidence: %w", err)
	}
	return nil
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
	analysisPath := filepath.Join(refined, "analysis.md")
	if handoff.HasHandoffMetadata(analysisPath) {
		if err := handoff.ValidateRefinedPackageForGate(refined, status.MissionID); err != nil {
			return domain.PipelineBypassDecision{}, fmt.Errorf("validate Archivist handoff: %w", err)
		}
	}
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

package mission

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	criticalhit "github.com/SergioLacerda/strategist-skill/internal/feats/critical_hit"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// executionEntryAction names the guarded transition in a bypass decision.
const executionEntryAction = "enter execution (handoff_challenge_satisfied)"

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
	stageRequest := domain.StageRequest{
		Route:            decision.SelectedRoute,
		PolicyVersion:    "route-resolution/v1",
		MissionID:        decision.MissionID,
		MissionExecution: true,
	}
	if decision.SelectedRoute == criticalhit.FeatID {
		// Scout remains the intake owner during the compatibility migration.
		// The route is retained only for downstream consumers; the domain
		// resolution is still a SHORT Stage with explicit Feat context.
		stageRequest.Role = "scout"
		stageRequest.Feat = criticalhit.FeatID
		stageRequest.MissionID = decision.MissionID
		stageRequest.CorrelationKey = decision.MissionID + ":" + criticalhit.FeatID
		stageRequest.GateRequired = true
	}
	resolution, err := domain.ResolveStage(stageRequest)
	if err != nil {
		return telemetry.RouteDecision{}, fmt.Errorf("resolve route Stage: %w", err)
	}
	artifact, err := domain.NewStageResolutionArtifact(resolution, decision.RequestCategory)
	if err != nil {
		return telemetry.RouteDecision{}, fmt.Errorf("build Stage resolution artifact: %w", err)
	}
	decision.Stage = string(artifact.Stage)
	decision.StageTrigger = artifact.Trigger
	decision.StageRole = artifact.Role
	decision.StageFeat = artifact.Feat
	decision.StageCorrelationID = artifact.CorrelationKey
	decision.StageGateRequired = artifact.GateRequired
	decision.StagePolicyVersion = artifact.PolicyVersion
	decision.StageReason = artifact.Reason
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
	decision, err := recordedRouteDecision(strategistRoot, status.MissionID)
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
	// Full-pipeline execution is authorized by the handoff challenge. A
	// Critical Hit SHORT execution is authorized by its explicit Stage gate and
	// enters the ordinary StateExecution state, so it must not be mistaken for
	// an unapproved DONE_ANALYSIS replay.
	gateApproved := executionGateApproved(status)
	stage := domain.Stage(decision.Stage)
	route := domain.PipelineRouteForScoutRoute(decision.SelectedRoute)
	if stage != "" {
		route = domain.PipelineRouteForStage(stage)
	}
	return domain.EvaluatePipelineBypass(domain.PipelineEvidence{
		Stage:              stage,
		Route:              route,
		BasePath:           basePath,
		MissionID:          status.MissionID,
		AttemptedAction:    executionEntryAction,
		DiscoveryPresent:   fileExists(filepath.Join(refined, "analysis.md")),
		RefinementPresent:  fileExists(filepath.Join(refined, "proposal.md")) && fileExists(filepath.Join(refined, "design.md")),
		TasksPresent:       fileExists(filepath.Join(refined, "tasks.md")),
		GatePresented:      gateApproved || status.StageGateRequired,
		GateApproved:       gateApproved,
		DirectGateApproved: gateApproved,
	}), nil
}

func executionGateApproved(status domain.MissionEngineStatus) bool {
	return status.State == domain.StateHandoffChallenge || status.StageGateApproved
}

// recordedRoute returns the selected_route Scout recorded for the mission, or ""
// when none was recorded.

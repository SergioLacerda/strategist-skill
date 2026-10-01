package mission

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

// requireAnalysisOnlyPackage rejects an analysis-only terminal event when the
// mission's refined tasks.md declares a documentation_target: those items must
// reach Sniper through the handoff challenge instead of being dropped.
func requireAnalysisOnlyPackage(basePath, missionID string, event domain.MissionEngineEvent) error {
	if !domain.MissionEventRequiresNoDocumentationTargets(event) {
		return nil
	}
	tasks := filepath.Join(basePath, "refined", missionID, "tasks.md")
	has, err := refinement.HasDocumentationTargets(tasks)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if has {
		return fmt.Errorf("mission submit: rejected: %s requires a package with no documentation_target, but %s declares one — use gate_approved, then verify the handoff and submit handoff_challenge_satisfied so Sniper materializes it", event, tasks)
	}
	return nil
}

// recordSniperClaimsOnHandoffPassed records a Sniper claim on each declared
// documentation_target once the mission has committed its transition into
// execution (event == handoff_challenge_satisfied, called after deps.Save
// succeeds and still inside the mission lock). Deriving claims here, rather
// than through a separate command the agent would have to remember to call,
// is the design.md Batch A decision for ADR-0057 § A1: the mechanism that
// must not be forgotten should not depend on being remembered. Every other
// event records nothing.
func recordSniperClaimsOnHandoffPassed(root, basePath, missionID string, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventHandoffSatisfied {
		return nil
	}
	if _, err := livemission.RecordSniperClaims(root, basePath, missionID, time.Now().UTC()); err != nil {
		return fmt.Errorf("record sniper claims: %w", err)
	}
	return nil
}

// requireExecutionEvidence is the live execution boundary: entering Sniper
// execution is rejected as pipeline_bypass_detected unless the evidence the
// mission's Scout route requires is present (see
// internal/mission.EvaluateExecutionEntry), and unless a durable, correlated
// handoff outcome authorizes this package: passed, or a skip the package's own
// facts authorize. The event name alone proves nothing. It returns the outcome
// to consume once the transition is committed. Every other event is unguarded,
// including gate_approved_analysis_only, which never enters execution.
func requireExecutionEvidence(root, basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (*handoff.Outcome, error) {
	if event != domain.MissionEventHandoffSatisfied {
		return nil, nil
	}
	decision, err := livemission.EvaluateExecutionEntry(root, basePath, status)
	if err != nil {
		return nil, fmt.Errorf("mission submit: %w", err)
	}
	if !decision.Allowed {
		return nil, fmt.Errorf("mission submit: rejected: %w", decision)
	}
	outcome, err := livemission.AuthorizeHandoffExecution(root, basePath, status)
	if err != nil {
		return nil, fmt.Errorf("mission submit: rejected: %w", err)
	}
	return &outcome, nil
}

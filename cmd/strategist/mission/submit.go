package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
)

// NewSubmit builds `mission submit`.
func NewSubmit(deps LifecycleDependencies) *cobra.Command {
	var event string
	cmd := &cobra.Command{Use: "submit", Short: "Submit one authoritative mission event"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.Flags().StringVar(&event, "event", "", "mission event to submit (required)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunSubmit(cmd, deps, f.root, f.missionID, event, f.asJSON)
	}
	return cmd
}

// RunSubmit applies one event through the restored domain.MissionEngine and
// persists the resulting status.
func RunSubmit(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID, event string, asJSON bool) error {
	root, engine, err := loadForSubmit(deps, rootInput, missionID, domain.MissionEngineEvent(event))
	if err != nil {
		return err
	}
	status, err := engine.Submit(domain.MissionEngineEvent(event))
	if err != nil {
		return fmt.Errorf("mission submit: rejected: %w", err)
	}
	if err := deps.Save(root, status); err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	return deps.WriteResult(cmd, asJSON, status)
}

// loadForSubmit validates the request, applies the pre-transition guards and
// restores the mission engine the event will be submitted to.
func loadForSubmit(deps LifecycleDependencies, rootInput, missionID string, event domain.MissionEngineEvent) (string, *domain.MissionEngine, error) {
	if err := deps.RequireMissionID(missionID); err != nil {
		return "", nil, err
	}
	if event == "" {
		return "", nil, fmt.Errorf("mission submit: --event is required")
	}
	root, basePath, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return "", nil, fmt.Errorf("mission submit: %w", err)
	}
	if err := requireAnalysisOnlyPackage(basePath, missionID, event); err != nil {
		return "", nil, err
	}
	engine, _, err := deps.Load(root, missionID)
	if err != nil {
		return "", nil, fmt.Errorf("mission submit: %w", err)
	}
	if err := requireExecutionEvidence(root, basePath, engine.Status(), event); err != nil {
		return "", nil, err
	}
	return root, engine, nil
}

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
		return fmt.Errorf("mission submit: rejected: %s requires a package with no documentation_target, but %s declares one — use gate_approved / handoff_challenge_passed so Sniper materializes it", event, tasks)
	}
	return nil
}

// requireExecutionEvidence is the live execution boundary: entering Sniper
// execution is rejected as pipeline_bypass_detected unless the evidence the
// mission's Scout route requires is present (see
// internal/mission.EvaluateExecutionEntry). Every other event is unguarded.
func requireExecutionEvidence(root, basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventHandoffPassed {
		return nil
	}
	decision, err := livemission.EvaluateExecutionEntry(root, basePath, status)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if !decision.Allowed {
		return fmt.Errorf("mission submit: rejected: %w", decision)
	}
	return nil
}

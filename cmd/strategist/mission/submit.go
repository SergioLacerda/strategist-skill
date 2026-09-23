package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	if event == "" {
		return fmt.Errorf("mission submit: --event is required")
	}
	root, basePath, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if err := requireAnalysisOnlyPackage(basePath, missionID, domain.MissionEngineEvent(event)); err != nil {
		return err
	}
	engine, _, err := deps.Load(root, missionID)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
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

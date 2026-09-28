package mission

import (
	"fmt"
	"io"

	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/spf13/cobra"
)

// NewRoute builds `mission route`, the live write path for Scout's
// route_decision: the JSON decision (scout-route-decision.schema.yaml) is read
// from stdin and appended to the route-decision history, where the execution
// boundary reads it back to decide which evidence the mission's route needs.
func NewRoute(deps LifecycleDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route",
		Short: "Record Scout's route decision for a mission (JSON on stdin)",
	}
	f := bindLifecycleFlags(cmd, deps)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunRoute(cmd, deps, f.root, f.missionID)
	}
	return cmd
}

// RunRoute validates and records one route decision. A second decision for the
// same mission is skipped, never overwritten.
func RunRoute(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID string) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission route: %w", err)
	}
	raw, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("mission route: read route decision: %w", err)
	}
	appended, err := livemission.RecordRouteDecision(root, missionID, raw)
	if err != nil {
		return fmt.Errorf("mission route: %w", err)
	}
	status := "recorded"
	if !appended {
		status = "already_recorded"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s route_decision=%s\n", missionID, status); err != nil {
		return fmt.Errorf("mission route: write result: %w", err)
	}
	return nil
}

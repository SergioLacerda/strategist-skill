package mission

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	criticalhit "github.com/SergioLacerda/strategist-skill/internal/feats/critical_hit"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
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
	appended, err := recordRoute(cmd, deps, root, missionID, raw)
	if err != nil {
		return fmt.Errorf("mission route: %w", err)
	}
	if err := activateSelectedCriticalHit(deps, root, missionID, raw); err != nil {
		return fmt.Errorf("mission route: activate Critical Hit: %w", err)
	}
	status := routeStatus(appended)
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "mission_id=%s route_decision=%s\n", missionID, status); err != nil {
		return fmt.Errorf("mission route: write result: %w", err)
	}
	return nil
}

func activateSelectedCriticalHit(deps LifecycleDependencies, root, missionID string, raw []byte) error {
	if selectedRoute(raw) != criticalhit.FeatID || deps.ActivateCriticalHit == nil {
		return nil
	}
	_, err := deps.ActivateCriticalHit(root, missionID)
	return err
}

func selectedRoute(raw []byte) string {
	var decision struct {
		SelectedRoute string `json:"selected_route"`
	}
	if err := json.Unmarshal(raw, &decision); err != nil {
		return ""
	}
	return decision.SelectedRoute
}

func recordRoute(cmd *cobra.Command, deps LifecycleDependencies, root, missionID string, raw []byte) (bool, error) {
	if deps.RecordRoute != nil {
		appended, err := application.RecordRoute(cmd.Context(), application.RecordRouteRequest{Root: root, MissionID: missionID, Raw: raw}, deps.RecordRoute)
		if err != nil {
			return false, fmt.Errorf("record route: %w", err)
		}
		return appended, nil
	}
	var sink telemetry.EventSink
	if deps.TelemetrySink != nil {
		sink = deps.TelemetrySink()
	}
	appended, err := livemission.RecordRouteDecisionWithTelemetry(cmd.Context(), root, missionID, raw, sink)
	if err != nil {
		return false, fmt.Errorf("record route with telemetry: %w", err)
	}
	return appended, nil
}

func routeStatus(appended bool) string {
	if appended {
		return "recorded"
	}
	return "already_recorded"
}

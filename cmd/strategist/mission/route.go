package mission

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
		Long: `Record Scout's route decision for a mission. The decision is one JSON object
read from stdin, shaped by schemas/scout-route-decision.schema.yaml (mission_id,
role, component, request_category, selected_route, route_reason,
route_confidence, evidence_state, discovery_subtype when full_pipeline needs
discovery, fallback_route, gate_required). The first recorded decision wins;
a replay is reported as already_recorded. intake_done is rejected until a
decision is recorded.

Input is UTF-8; a leading UTF-8 byte order mark is tolerated.`,
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
	raw, err := cliutil.ReadInput(cmd.InOrStdin())
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
	before := statusBeforeActivation(deps, root, missionID)
	after, err := deps.ActivateCriticalHit(root, missionID)
	if err != nil {
		return err
	}
	if before.Phase != "" {
		markers := domain.LifecycleMarkersForTransition(domain.MissionEventCriticalHitIntent, before, after)
		recordLifecycleBestEffort(deps, root, missionID, markers)
	}
	return nil
}

// statusBeforeActivation reads the mission status the activation will leave, so
// the lifecycle markers can be derived. A load failure yields the zero status
// and simply skips the markers; it never blocks the activation.
func statusBeforeActivation(deps LifecycleDependencies, root, missionID string) domain.MissionEngineStatus {
	if deps.Load == nil {
		return domain.MissionEngineStatus{}
	}
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return domain.MissionEngineStatus{}
	}
	return status
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

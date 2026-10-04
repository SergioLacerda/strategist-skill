package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ValidateRouteDecisionLine parses a single JSON line and checks required
// fields and allowed values per scout-route-decision.schema.yaml.
func ValidateRouteDecisionLine(line string) error {
	var d RouteDecision
	if err := json.Unmarshal([]byte(line), &d); err != nil {
		return fmt.Errorf("route decision line is not valid JSON: %w", err)
	}
	var errs []error
	errs = append(errs, requiredRouteField("mission_id", d.MissionID)...)
	errs = append(errs, requiredRouteField("request_category", d.RequestCategory)...)
	errs = append(errs, allowedRouteValue("selected_route", d.SelectedRoute, allowedSelectedRoutes)...)
	errs = append(errs, requiredRouteField("route_reason", d.RouteReason)...)
	errs = append(errs, routeConfidenceRange(d.RouteConfidence)...)
	errs = append(errs, allowedRouteValue("evidence_state", d.EvidenceState, allowedEvidenceStates)...)
	errs = append(errs, fallbackRouteValue(d.FallbackRoute)...)
	errs = append(errs, stageProjectionValue(d)...)
	errs = append(errs, requiredRouteField("timestamp", d.Timestamp)...)
	return errors.Join(errs...)
}

func stageProjectionValue(d RouteDecision) []error {
	if d.Stage == "" {
		return nil // legacy history predates the canonical Stage projection
	}
	stage := domain.Stage(d.Stage)
	if err := stage.Validate(); err != nil {
		return []error{err}
	}
	var errs []error
	errs = append(errs, requiredRouteField("stage_trigger", d.StageTrigger)...)
	errs = append(errs, requiredRouteField("stage_policy_version", d.StagePolicyVersion)...)
	errs = append(errs, requiredRouteField("stage_reason", d.StageReason)...)
	return errs
}

func requiredRouteField(name, value string) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	return nil
}

func allowedRouteValue(name, value string, allowed map[string]bool) []error {
	if value == "" {
		return []error{fmt.Errorf("%s is required", name)}
	}
	if !allowed[value] {
		return []error{fmt.Errorf("%s %q is not an allowed value", name, value)}
	}
	return nil
}

func routeConfidenceRange(confidence float64) []error {
	if confidence < 0.0 || confidence > 1.0 {
		return []error{fmt.Errorf("route_confidence %v is out of range [0.0, 1.0]", confidence)}
	}
	return nil
}

func fallbackRouteValue(fallbackRoute string) []error {
	if fallbackRoute != "" && fallbackRoute != "full_pipeline" {
		return []error{fmt.Errorf("fallback_route %q must be full_pipeline", fallbackRoute)}
	}
	return nil
}

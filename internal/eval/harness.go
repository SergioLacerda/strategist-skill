package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// RunScenario dispatches to the target domain/treasure function named by
// s.Input.Target and evaluates s.Expected plus s.Assertions against the
// result. No provider is invoked — the function under test is called
// directly, in-process.
func RunScenario(s Scenario) ScenarioResult {
	start := time.Now()
	res := ScenarioResult{ScenarioID: s.ID}

	switch s.Input.Target {
	case TargetStateMachine:
		runStateMachineScenario(s, &res)
	case TargetRouteDecision:
		runRouteDecisionScenario(s, &res)
	case TargetScopeFilter:
		runScopeFilterScenario(s, &res)
	case TargetSlotWriteScope:
		runSlotWriteScopeScenario(s, &res)
	case TargetArtifactCheck:
		runArtifactCheckScenario(s, &res)
	case TargetChestGrade:
		runChestGradeScenario(s, &res)
	case TargetJewelTrust:
		runJewelTrustScenario(s, &res)
	default:
		res.Violations = append(res.Violations, Violation{Message: fmt.Sprintf("unknown target %q", s.Input.Target)})
	}

	applyGenericAssertions(s, &res)

	res.Duration = time.Since(start)
	res.Passed = len(res.Violations) == 0
	return res
}

// applyGenericAssertions evaluates assertion types that are meaningful
// regardless of Target. Target-specific assertion types (required-event,
// scope-includes, scope-excludes) are evaluated inline by their owning
// run*Scenario function instead, since they need target-local data.
func applyGenericAssertions(s Scenario, res *ScenarioResult) {
	for _, a := range s.Assertions {
		if a.Type != AssertArtifactExists {
			continue
		}
		if _, err := os.Stat(filepath.Clean(a.Value)); err != nil {
			res.Violations = append(res.Violations, Violation{
				AssertionType: AssertArtifactExists,
				Message:       "artifact does not exist",
				Expected:      a.Value,
				Actual:        err.Error(),
			})
		}
	}
}

func runStateMachineScenario(s Scenario, res *ScenarioResult) {
	start := domain.MissionState(paramString(s.Input.Params, "start"))
	eventStrs := paramStringSlice(s.Input.Params, "events")
	events := make([]domain.TransitionEvent, 0, len(eventStrs))
	for _, e := range eventStrs {
		events = append(events, domain.TransitionEvent(e))
	}

	final := domain.RunStateMachine(start, events)
	if s.Expected.State != "" && string(final) != s.Expected.State {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertEqualState,
			Message:       "final state mismatch",
			Expected:      s.Expected.State,
			Actual:        string(final),
		})
	}

	for _, a := range s.Assertions {
		if a.Type == AssertRequiredEvent && !containsString(eventStrs, a.Value) {
			res.Violations = append(res.Violations, Violation{
				AssertionType: AssertRequiredEvent,
				Message:       "required event not present in input events",
				Expected:      a.Value,
				Actual:        fmt.Sprintf("%v", eventStrs),
			})
		}
	}
}

func runRouteDecisionScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	route := paramString(p, "route")
	metadata := domain.RouteRequestMetadata{
		HasContext:           paramBool(p, "has_context"),
		TouchesSourceCode:    paramBool(p, "touches_source_code"),
		AnalysisArtifactOnly: paramBool(p, "analysis_artifact_only"),
		RequiresDiscovery:    paramBool(p, "requires_discovery"),
	}

	decision := domain.ValidateRouteDecision(route, metadata)
	checkStatusAndReason(res, string(decision.Status), decision.Reason, s.Expected)
}

func runSlotWriteScopeScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	scope := domain.SlotWriteScope{
		SlotName:      paramString(p, "slot_name"),
		AllowedPrefix: paramString(p, "allowed_prefix"),
		AllowedExt:    paramString(p, "allowed_ext"),
	}
	attemptedPath := paramString(p, "attempted_path")

	err := domain.ValidateSlotWrite(scope, attemptedPath)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func runChestGradeScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	chestID := paramString(p, "chest_id")
	grade := domain.ChestGrade{
		SourceGrade:          paramString(p, "source_grade"),
		ReuseValue:           paramString(p, "reuse_value"),
		ImplementationStatus: paramString(p, "implementation_status"),
	}

	err := domain.ValidateChestGrade(chestID, grade)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func runJewelTrustScenario(s Scenario, res *ScenarioResult) {
	p := s.Input.Params
	jewelID := paramString(p, "jewel_id")
	jewelTrust := paramString(p, "jewel_trust")
	chestTrustTier := paramString(p, "chest_trust_tier")

	err := domain.ValidateJewelTrust(jewelID, jewelTrust, chestTrustTier)
	actualStatus, actualReason := "allowed", ""
	if err != nil {
		actualStatus, actualReason = "blocked", err.Error()
	}
	checkStatusAndReason(res, actualStatus, actualReason, s.Expected)
}

func checkStatusAndReason(res *ScenarioResult, actualStatus, actualReason string, expected Expected) {
	if expected.Status != "" && actualStatus != expected.Status {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertEqualState,
			Message:       "status mismatch",
			Expected:      expected.Status,
			Actual:        actualStatus,
		})
	}
	if expected.Reason != "" && !strings.Contains(actualReason, expected.Reason) {
		res.Violations = append(res.Violations, Violation{
			AssertionType: AssertEqualState,
			Message:       "reason does not contain expected substring",
			Expected:      expected.Reason,
			Actual:        actualReason,
		})
	}
}

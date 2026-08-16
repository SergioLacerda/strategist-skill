package eval

import (
	"fmt"
	"os"
	"path/filepath"
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
	case TargetCriticalHitTrigger:
		runCriticalHitTriggerScenario(s, &res)
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

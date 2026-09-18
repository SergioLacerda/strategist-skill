//go:build eval

// Terminal-boundary scenarios for the explicit Approval Gate and Sniper route.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestSniperAcceptedGateReachesTerminalDelivery(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "sniper-accepted-gate-terminal-delivery",
		Description: "an accepted gate and passed handoff reach Sniper and terminate at delivery",
		Input: eval.Input{Target: eval.TargetStateMachine, Params: map[string]any{
			"start":  "APPROVAL_GATE",
			"events": []any{"gate_approved", "handoff_challenge_passed", "sniper_done"},
		}},
		Expected: eval.Expected{State: "DONE_DELIVERY"},
		Assertions: []eval.Assertion{
			{Type: eval.AssertRequiredEvent, Value: "gate_approved"},
			{Type: eval.AssertRequiredEvent, Value: "handoff_challenge_passed"},
			{Type: eval.AssertRequiredEvent, Value: "sniper_done"},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

func TestSniperRejectedGateDoesNotReachExecution(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "sniper-rejected-gate-no-execution",
		Description: "a rejected gate terminates analysis before handoff or Sniper execution",
		Input: eval.Input{Target: eval.TargetStateMachine, Params: map[string]any{
			"start":  "APPROVAL_GATE",
			"events": []any{"gate_denied"},
		}},
		Expected:   eval.Expected{State: "DONE_ANALYSIS"},
		Assertions: []eval.Assertion{{Type: eval.AssertRequiredEvent, Value: "gate_denied"}},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

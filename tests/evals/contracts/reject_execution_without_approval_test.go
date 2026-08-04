//go:build eval

// Scenario A1: no execution without approval.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestA1_RejectExecutionWithoutApproval(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "reject-execution-without-approval",
		Description: "the skill must refuse execution when the approval gate denies the mission",
		Input: eval.Input{
			Target: eval.TargetStateMachine,
			Params: map[string]any{
				"start":  "APPROVAL_GATE",
				"events": []any{"gate_denied"},
			},
		},
		Expected: eval.Expected{State: "DONE_ANALYSIS"},
		Assertions: []eval.Assertion{
			{Type: eval.AssertRequiredEvent, Value: "gate_denied"},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

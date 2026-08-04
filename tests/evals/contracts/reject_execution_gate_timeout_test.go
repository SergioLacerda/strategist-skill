//go:build eval

// Scenario A2: execution blocked on gate timeout.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestA2_RejectExecutionGateTimeout(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "reject-execution-gate-timeout",
		Description: "a gate timeout resolves as analysis_delivered, not execution",
		Input: eval.Input{
			Target: eval.TargetStateMachine,
			Params: map[string]any{
				"start":  "APPROVAL_GATE",
				"events": []any{"gate_timeout"},
			},
		},
		Expected: eval.Expected{State: "DONE_ANALYSIS"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

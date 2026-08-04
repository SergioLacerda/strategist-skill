//go:build eval

// Scenario A3: execution proceeds after explicit gate approval.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestA3_AcceptExecutionAfterApproval(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "accept-execution-after-approval",
		Description: "gate_approved transitions APPROVAL_GATE to EXECUTION",
		Input: eval.Input{
			Target: eval.TargetStateMachine,
			Params: map[string]any{
				"start":  "APPROVAL_GATE",
				"events": []any{"gate_approved"},
			},
		},
		Expected: eval.Expected{State: "EXECUTION"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

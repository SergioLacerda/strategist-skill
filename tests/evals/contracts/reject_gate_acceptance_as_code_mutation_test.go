//go:build eval

// Scenario A5: gate acceptance is not code mutation approval —
// internal/domain.ValidateRouteDecision blocks direct_execute whenever the
// request touches source code, independent of any gate outcome.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestA5_RejectGateAcceptanceAsCodeMutation(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "reject-gate-acceptance-as-code-mutation",
		Description: "direct_execute is blocked whenever the request touches source code, regardless of gate acceptance",
		Input: eval.Input{
			Target: eval.TargetRouteDecision,
			Params: map[string]any{
				"route":               "direct_execute",
				"has_context":         true,
				"touches_source_code": true,
			},
		},
		Expected: eval.Expected{Status: "blocked", Reason: "cannot mutate source"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

//go:build eval

// Scenario A4: implementation_handoff items (code/test files) are never
// Sniper-writable — internal/domain.ValidateSlotWrite is the deterministic
// enforcement mechanism behind that rule.
package contracts_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestA4_RejectImplementationHandoffAsSniperTask(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "reject-implementation-handoff-as-sniper-task",
		Description: "Sniper's execution slot cannot write a .go file — only its declared documentation prefix/extension",
		Input: eval.Input{
			Target: eval.TargetSlotWriteScope,
			Params: map[string]any{
				"slot_name":      "execution",
				"allowed_prefix": ".analysis/archived",
				"allowed_ext":    ".md",
				"attempted_path": "internal/eval/scenario.go",
			},
		},
		Expected: eval.Expected{Status: "blocked", Reason: "slot_write_scope_violation"},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

//go:build eval

// Group C: treasure-chest scope filtering — a chest scoped to "execution"
// must not be selected during discovery, a chest scoped to "discovery" or
// "all" must be. internal/treasure.FilterRowsByScope is the deterministic
// enforcement mechanism behind this rule (see roles/ranger.yaml's
// consult_treasure_chests / roles/archivist.yaml's equivalent, both scoped
// per-slot).
package scenarios_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/eval"
	"github.com/stretchr/testify/assert"
)

func TestC1_RangerIgnoresExecutionScopeChest_UsesDiscoveryScopeChest(t *testing.T) {
	res := eval.RunScenario(eval.Scenario{
		ID:          "ranger-uses-discovery-scope-ignores-execution-scope",
		Description: "filtering by 'discovery' selects discovery- and all-scoped chests, excludes execution-only chests",
		Input: eval.Input{
			Target: eval.TargetScopeFilter,
			Params: map[string]any{
				"rows": []any{
					map[string]any{"id": "runbooks", "scope": []any{"all"}},
					map[string]any{"id": "discovery-notes", "scope": []any{"discovery"}},
					map[string]any{"id": "execution-only", "scope": []any{"execution"}},
				},
				"value": "discovery",
			},
		},
		Expected: eval.Expected{IDs: []string{"runbooks", "discovery-notes"}},
		Assertions: []eval.Assertion{
			{Type: eval.AssertScopeIncludes, Value: "discovery-notes"},
			{Type: eval.AssertScopeIncludes, Value: "runbooks"},
			{Type: eval.AssertScopeExcludes, Value: "execution-only"},
		},
	})
	assert.True(t, res.Passed, "violations: %+v", res.Violations)
}

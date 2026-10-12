package calibration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const realPlan = `## 1. Mixed grammars

- [x] 1.1 [task_type: implementation_handoff] Add the check in internal/check/x.go; verify the new test passes.
- [ ] [analysis_artifact] Compare the inventory
  against the source and record each mismatch; verify the table exists.
- [x] A.1 Add the detection
  (new function in cmd/strategist/x.go), then wire it. Done: wired.

- [x] 2.1 [implementation_handoff] Old grammar task; verify it.
`

func TestParseTasksReadsEveryGrammarAndWrappedItems(t *testing.T) {
	items := parseTasks(realPlan)

	require.Len(t, items, 4)
	assert.Equal(t, []string{"implementation_handoff", "analysis_artifact", "", "implementation_handoff"}, []string{items[0].Type, items[1].Type, items[2].Type, items[3].Type})
	assert.Equal(t, items[1].Start+1, items[1].End, "the wrapped line belongs to its item")
	assert.Equal(t, items[2].Start+1, items[2].End)
	assert.Len(t, typedTasks(realPlan), 3, "the operators rewrite only typed tasks")
}

func TestTheLocalChecksSeeRealPackagesTheOldGrammarMissed(t *testing.T) {
	plan := Fields{"implementation_plan": realPlan, "approved_scope": "- allowed: docs/**", "acceptance_checks": "- passes"}
	assert.ElementsMatch(t, []string{"task_untyped", "task_unverified"}, LocalFlags(BoundaryPackage, plan),
		"the unnumbered, untyped and wrapped task without a verification is flagged")

	conforming := Fields{"implementation_plan": "- [x] [task_type: implementation_handoff] Build it\n  in two lines; verify it passes.\n- [ ] [analysis_artifact] Note it; verify it exists.",
		"approved_scope": "- allowed: docs/**", "acceptance_checks": "- passes"}
	assert.Empty(t, LocalFlags(BoundaryPackage, conforming))
}

func TestKindMismatchReadsTheWholeWrappedTask(t *testing.T) {
	plan := Fields{"implementation_plan": "- [ ] [analysis_artifact] Review the design\n  and change cmd/strategist/main.go; verify it.", "approved_scope": "x", "acceptance_checks": "x"}
	assert.Contains(t, LocalFlags(BoundaryPackage, plan), "task_kind_mismatch")
}

func TestOperatorsRewriteTheNewGrammarsAndLeaveTheOldOneAsItWas(t *testing.T) {
	ctx := Context{Seed: 1, Key: "k", Level: 1}
	stripped := stripTaskTypes(Fields{"implementation_plan": realPlan}, ctx)["implementation_plan"]
	assert.NotContains(t, stripped, "[task_type: implementation_handoff]")
	assert.NotContains(t, stripped, "[analysis_artifact]")
	assert.Contains(t, stripped, "- [x] 2.1  Old grammar task", "the old grammar keeps its exact layout, double space included")

	relabeled := relabelImplementation(Fields{"implementation_plan": realPlan}, ctx)["implementation_plan"]
	assert.Contains(t, relabeled, "[task_type: analysis_artifact] Add the check")
	assert.Contains(t, relabeled, "2.1 [analysis_artifact] Old grammar task")

	dropped := dropVerification(Fields{"implementation_plan": realPlan}, ctx)["implementation_plan"]
	assert.NotContains(t, strings.ToLower(dropped), "verify the table exists", "a verification on a wrapped task is removed with its continuation")
	assert.Contains(t, dropped, "Compare the inventory", "the part before the verification stays")
}

func TestAnOperatorThatAppliesToARealPlanIsOfferedForIt(t *testing.T) {
	mixed := Fields{"implementation_plan": realPlan}
	assert.True(t, tasksPresent(mixed))
	assert.True(t, verifiedTasks(mixed))
	assert.True(t, planHas(relabelTag)(mixed), "a [task_type: implementation_handoff] plan can be relabelled")
	assert.False(t, tasksPresent(Fields{"implementation_plan": "no tasks here"}))
}

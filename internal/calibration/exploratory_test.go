package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcludedOperatorsLeaveTheMetricsAndAreNamedInTheReport(t *testing.T) {
	obs := runAll(t, corpus(60), FuncJudge(ruleJudge), 2)
	full := Evaluate(obs, DefaultConfig())
	cfg := DefaultConfig()
	cfg.Filter.ExcludedOperators = map[string]bool{"strip_task_types": true, "drop_task_verification": true}
	partial := Evaluate(obs, cfg)

	assert.Empty(t, full.Exploratory)
	assert.Equal(t, []string{"drop_task_verification", "strip_task_types"}, partial.Exploratory)
	assert.Less(t, partial.Counts.Defects, full.Counts.Defects)
	for _, op := range partial.Operators {
		assert.NotEqual(t, "strip_task_types", op.Operator)
	}
}

func TestACalibrationIsNeverPublishedFromAWhatIf(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Filter.ExcludedOperators = map[string]bool{"strip_task_types": true}
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), cfg)
	_, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: "p", ProfileVersion: "v", Now: fixedNow})
	require.ErrorContains(t, err, "exploratory")
}

func TestARecordKeepsTheOperatorListTheCaseRestrictedItselfTo(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), DefaultConfig())
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: "p", ProfileVersion: "v", Operators: []string{"drop_approved_scope", "truncate_plan"}, Now: fixedNow})
	require.NoError(t, err)
	assert.Equal(t, []string{"drop_approved_scope", "truncate_plan"}, record.OperatorFilter)
	require.NoError(t, record.Validate())
	plain, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: "p", ProfileVersion: "v", Now: fixedNow})
	require.NoError(t, err)
	assert.Empty(t, plain.OperatorFilter)
}

func TestStripAllTaskTypesIsAFullLevelOperatorAndTheGradedOneStays(t *testing.T) {
	all, ok := ByName("strip_all_task_types")
	require.True(t, ok)
	assert.Empty(t, all.Levels)
	graded, ok := ByName("strip_task_types")
	require.True(t, ok)
	assert.Len(t, graded.Levels, 3)

	src := source(1)
	var full Case
	for _, c := range Generate(src, Options{Seed: 3, Operators: []string{"strip_all_task_types"}}) {
		if c.Operator == "strip_all_task_types" {
			full = c
		}
	}
	require.NotEmpty(t, full.ID)
	assert.Equal(t, ExpectReject, full.Expect)
	assert.Empty(t, taskIndexes(full.Fields["implementation_plan"]), "every task lost its type")
}

func TestTheMonotonicDetailNamesTheOperatorThatInverts(t *testing.T) {
	obs := ladderObservations(0.70, []float64{0.20, 0.60, 0.15}, 0.01)
	gate := gateOf(t, obs, DefaultLimits())
	assert.Contains(t, gate.Detail, "inverted in strip_task_types (1)")
}

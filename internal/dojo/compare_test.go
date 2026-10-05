package dojo_test

import (
	"bytes"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func observationsFor(t *testing.T, judge calibration.FuncJudge) []calibration.Observation {
	t.Helper()
	w := goldenWorkspace(t, 40)
	in := runInput(t, w, judge, nil)
	_, err := dojo.RunCalibration(t.Context(), in)
	require.NoError(t, err)
	observed, _, err := dojo.ReadObservations(in.Paths.ObservationsPath)
	require.NoError(t, err)
	return observed
}

func TestCompareShowsWhatAJudgeChangeDidPerOperator(t *testing.T) {
	blind := calibration.FuncJudge(func(b string, f calibration.Fields) calibration.Verdict {
		if f["approved_scope"] == "" {
			return calibration.ScoredVerdict(0.99)
		}
		return ruleJudge(b, f)
	})
	a := observationsFor(t, blind)
	b := observationsFor(t, calibration.FuncJudge(ruleJudge))

	cmp := dojo.Compare("blind", "sees", a, b)

	var scope dojo.CompareRow
	for _, row := range cmp.Rows {
		if row.Operator == "drop_approved_scope" {
			scope = row
		}
	}
	assert.Greater(t, scope.A, 0.8)
	assert.Less(t, scope.B, 0.5)
	assert.Greater(t, cmp.AUCB, cmp.AUCA)
	require.Len(t, cmp.SweepA, len(cmp.SweepB))

	var out bytes.Buffer
	require.NoError(t, cmp.Render(&out))
	for _, want := range []string{"laboratory comparison", "no calibration, no recommendation", "drop_approved_scope", "(intact golden package)", "AUC", "exploratory operating points"} {
		assert.Contains(t, out.String(), want)
	}
}

func TestCompareHandlesAnOperatorOnOneSideOnly(t *testing.T) {
	a := observationsFor(t, calibration.FuncJudge(ruleJudge))
	cmp := dojo.Compare("a", "b", a, nil)
	assert.NotEmpty(t, cmp.Rows)
	for _, row := range cmp.Rows {
		assert.Zero(t, row.NB)
	}
}

func TestAnExploratoryCheckIsLabelledAndLeavesTheCaseUntouched(t *testing.T) {
	w := goldenWorkspace(t, 40)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	_, err := dojo.RunCalibration(t.Context(), in)
	require.NoError(t, err)

	out, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: 0.8, Session: "latest", ExcludeOperators: []string{"strip_task_types"}})
	require.NoError(t, err)
	assert.Equal(t, "calibration exploratory what-if", out.Result.Items[0].Label)
	assert.Equal(t, []string{"strip_task_types"}, out.Report.Exploratory)
}

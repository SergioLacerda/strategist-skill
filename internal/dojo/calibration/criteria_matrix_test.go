package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoredStat(operator string, scores ...map[string]float64) CaseStat {
	stat := CaseStat{ID: operator, Operator: operator}
	for _, s := range scores {
		stat.Observations = append(stat.Observations, Observation{Scores: s})
	}
	return stat
}

func TestTheCriteriaMatrixAveragesEachCriterionPerGroup(t *testing.T) {
	stats := []CaseStat{
		scoredStat("", map[string]float64{"scope": 0.9, "plan": 0.8}),
		scoredStat("drop_approved_scope", map[string]float64{"scope": 0.1, "plan": 0.8}, map[string]float64{"scope": 0.3, "plan": 0.6}),
	}

	matrix := BuildCriteriaMatrix(stats)

	require.NotNil(t, matrix)
	assert.Equal(t, []string{"plan", "scope"}, matrix.Criteria)
	require.Len(t, matrix.Rows, 2)
	assert.Equal(t, GoldenRow, matrix.Rows[0].Group, "the intact packages come first")
	assert.InDelta(t, 0.2, matrix.Rows[1].Means["scope"], 1e-9)
	assert.InDelta(t, 0.7, matrix.Rows[1].Means["plan"], 1e-9)
	assert.Equal(t, 2, matrix.Rows[1].Runs)
}

func TestTheCriteriaMatrixIsAbsentForAnAggregateProfile(t *testing.T) {
	assert.Nil(t, BuildCriteriaMatrix([]CaseStat{scoredStat("", nil)}))
}

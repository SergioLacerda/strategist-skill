package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func atomicObs(scores map[string]float64) Observation {
	return Observation{CaseID: "c", Decision: "reject", PAccept: 0.1, Scores: scores}
}

func TestWithoutCriteriaRereadsTheLowestRemainingAnswer(t *testing.T) {
	obs := atomicObs(map[string]float64{"a": 0.1, "b": 0.7, "c": 0.9})

	out, err := WithoutCriteria([]Observation{obs}, []string{"a"})

	require.NoError(t, err)
	assert.InDelta(t, 0.7, out[0].PAccept, 1e-9)
	assert.Equal(t, "accept", out[0].Decision)
	assert.InDelta(t, 0.3, out[0].Probabilities["reject"], 1e-9)
	assert.NotContains(t, out[0].Scores, "a")
	assert.InDelta(t, 0.1, obs.PAccept, 1e-9, "the stored observation is not changed")
	assert.Contains(t, obs.Scores, "a")
}

func TestWithoutCriteriaRejectsBelowTheAcceptLine(t *testing.T) {
	out, err := WithoutCriteria([]Observation{atomicObs(map[string]float64{"a": 0.9, "b": 0.3})}, []string{"a"})
	require.NoError(t, err)
	assert.Equal(t, "reject", out[0].Decision)
}

func TestWithoutCriteriaRefusesATypoAndAnEmptyRemainder(t *testing.T) {
	obs := []Observation{atomicObs(map[string]float64{"a": 0.9})}
	_, err := WithoutCriteria(obs, []string{"typo"})
	require.ErrorContains(t, err, "no observation answered")
	_, err = WithoutCriteria(obs, []string{"a"})
	require.ErrorContains(t, err, "leaves none")
}

func TestWithoutCriteriaLeavesAnAggregateProfileAlone(t *testing.T) {
	obs := []Observation{{CaseID: "c", Decision: "accept", PAccept: 0.8}}
	out, err := WithoutCriteria(obs, nil)
	require.NoError(t, err)
	assert.Equal(t, obs, out)
}

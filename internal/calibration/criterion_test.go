package calibration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoredObs(id, parent, op string, kind Kind, golden bool, scores map[string]float64) []Observation {
	var out []Observation
	for run := 1; run <= 2; run++ {
		out = append(out, Observation{CaseID: id, ParentID: parent, Kind: kind, Operator: op, Expect: rejectOf(kind), Golden: golden,
			Run: run, Decision: "accept", PAccept: 0.5, Scores: scores})
	}
	return out
}

func rejectOf(kind Kind) Expect {
	if kind == KindGolden {
		return ExpectAccept
	}
	return ExpectReject
}

// criterionCorpus has golden parents that answer the acceptance question well and the
// verification question across the whole range, and defects of the operators that target them.
func criterionCorpus(parents int) []Observation {
	var obs []Observation
	for i := 0; i < parents; i++ {
		id := fmt.Sprintf("p%02d", i)
		golden := map[string]float64{"acceptance_checkable": 0.8 + float64(i%5)/50, "validation_stated": 0.1 + float64(i%9)/10}
		obs = append(obs, scoredObs(id, "", "", KindGolden, true, golden)...)
		obs = append(obs, scoredObs(id+"~empty", id, "empty_acceptance_checks", KindSynthetic, false,
			map[string]float64{"acceptance_checkable": 0.06, "validation_stated": golden["validation_stated"]})...)
		obs = append(obs, scoredObs(id+"~drop", id, "truncate_plan", KindSynthetic, false,
			map[string]float64{"acceptance_checkable": golden["acceptance_checkable"], "validation_stated": golden["validation_stated"] - 0.05})...)
	}
	return obs
}

func TestACriterionThatSeparatesIsReliableAndOneThatOverlapsIsNot(t *testing.T) {
	stats := Stats(criterionCorpus(30), Filter{})

	results := CriterionCalibration(stats, DefaultPolicy(), DefaultLimits())

	byName := map[string]CriterionResult{}
	for _, r := range results {
		byName[r.Criterion] = r
	}
	acceptance := byName["acceptance_checkable"]
	assert.Equal(t, CriterionReliable, acceptance.Status, "%+v", acceptance)
	assert.Equal(t, []string{"empty_acceptance_checks"}, acceptance.Operators)
	assert.InDelta(t, 1.0, acceptance.AUC, 1e-9)
	assert.Zero(t, acceptance.FalseFlag.K)
	assert.Equal(t, acceptance.Detection.N, acceptance.Detection.K)

	validation := byName["validation_stated"]
	assert.Equal(t, CriterionUnreliable, validation.Status, "the defect scores sit inside the golden range: %+v", validation)
	assert.NotEmpty(t, validation.Note)
}

func TestACriterionWithTooFewGoldenParentsIsNotMeasured(t *testing.T) {
	results := CriterionCalibration(Stats(criterionCorpus(6), Filter{}), DefaultPolicy(), DefaultLimits())
	require.NotEmpty(t, results)
	for _, r := range results {
		assert.Equal(t, CriterionNotMeasured, r.Status, r.Criterion)
	}
}

func TestAnAggregateProfileHasNoCriterionCalibration(t *testing.T) {
	obs := criterionCorpus(30)
	for i := range obs {
		obs[i].Scores = nil
	}
	assert.Empty(t, CriterionCalibration(Stats(obs, Filter{}), DefaultPolicy(), DefaultLimits()))
}

func TestTheDetectionBoundUsesTheEffectiveSampleAndRoundsDown(t *testing.T) {
	plain := NewRate(20, 20)
	assert.Greater(t, detectionLower(plain), detectionLower(NewClusterRate(20, 20, 8)), "fewer effective cases give a lower bound further down")
	assert.Zero(t, detectionLower(Rate{}))
}

func TestTheEvaluationAndTheRecordCarryTheCriterionCalibration(t *testing.T) {
	report := Evaluate(criterionCorpus(30), ConfigForRule(RuleGates3))
	require.NotEmpty(t, report.CriterionCalibration)
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: ProfileAtomicV1, ProfileVersion: "v", Now: fixedNow})
	require.NoError(t, err)
	assert.NotEmpty(t, record.CriterionCalibration)
	assert.NoError(t, record.Validate())
}

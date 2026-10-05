package calibration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ladderObservations builds one golden parent and one graded operator with the given
// mean score per level, each case observed in three runs spread by +/- jitter.
func ladderObservations(parent float64, levels []float64, jitter float64) []Observation {
	var out []Observation
	add := func(id, parentID string, kind Kind, level, score float64) {
		for run, delta := range []float64{-jitter, 0, jitter} {
			out = append(out, Observation{CaseID: id, ParentID: parentID, Kind: kind, Operator: opName(parentID), Level: level, Expect: expectOf(kind), Golden: true, Run: run + 1, Decision: "accept", PAccept: score + delta})
		}
	}
	add("p", "", KindGolden, 0, parent)
	for i, score := range levels {
		add(fmt.Sprintf("p~strip@%d", i), "p", KindBoundary, float64(i+1)/float64(len(levels)), score)
	}
	return out
}

func opName(parent string) string {
	if parent == "" {
		return ""
	}
	return "strip_task_types"
}

func expectOf(kind Kind) Expect {
	if kind == KindGolden {
		return ExpectAccept
	}
	return ExpectDrop
}

func gateOf(t *testing.T, observations []Observation, limits Limits) Gate {
	t.Helper()
	for _, g := range FitnessGates(Stats(observations, Filter{}), limits) {
		if g.Name == GateMonotonic {
			return g
		}
	}
	t.Fatal("no monotonic gate")
	return Gate{}
}

func TestARiseWithinTheMeasuredNoiseIsNotAnInversion(t *testing.T) {
	obs := ladderObservations(0.70, []float64{0.50, 0.58, 0.45}, 0.08) // a 0.08 rise, noise about 0.065
	strict := Gates1Limits()
	strict.NoiseFactor = 0
	assert.Equal(t, GateFailed, gateOf(t, obs, strict).Status, "without a noise term the 0.08 rise is an inversion")
	assert.Equal(t, GatePassed, gateOf(t, obs, Gates1Limits()).Status, "with it, the rise cannot be told from the judge's own variation")
}

func TestARiseBeyondTheNoiseStillFails(t *testing.T) {
	obs := ladderObservations(0.70, []float64{0.20, 0.60, 0.15}, 0.01)
	gate := gateOf(t, obs, Gates1Limits())
	assert.Equal(t, GateFailed, gate.Status)
	assert.InDelta(t, 1, gate.Value, 1e-9)
}

func TestAFlatLadderIsALimitationReportedNotAnInversion(t *testing.T) {
	obs := ladderObservations(0.70, []float64{0.69, 0.70, 0.68}, 0.01)
	gate := gateOf(t, obs, Gates1Limits())
	assert.Equal(t, GatePassed, gate.Status)
	assert.Contains(t, gate.Detail, "1 flat")
}

func TestAResponsiveLadderIsNotFlat(t *testing.T) {
	obs := ladderObservations(0.80, []float64{0.60, 0.40, 0.10}, 0.01)
	gate := gateOf(t, obs, Gates1Limits())
	assert.Equal(t, GatePassed, gate.Status)
	assert.Contains(t, gate.Detail, "0 flat")
}

func TestTheToleranceIsNeverBelowTheConfiguredOne(t *testing.T) {
	obs := ladderObservations(0.70, []float64{0.50, 0.58, 0.45}, 0.0)
	gate := gateOf(t, obs, Gates1Limits())
	assert.Contains(t, gate.Detail, "tolerance 0.050")
	assert.Equal(t, GateFailed, gate.Status, "a 0.08 rise with no noise at all is a real inversion")
}

// manyLadders builds n parents, each with the same graded ladder.
func manyLadders(n int, parent float64, levels []float64, jitter float64) []Observation {
	var out []Observation
	for i := 0; i < n; i++ {
		for _, o := range ladderObservations(parent, levels, jitter) {
			o.CaseID = fmt.Sprintf("%s-%d", o.CaseID, i)
			if o.ParentID != "" {
				o.ParentID = fmt.Sprintf("%s-%d", o.ParentID, i)
			}
			out = append(out, o)
		}
	}
	return out
}

func TestUnderGates2ARateOfInversionsPassesWhenItsUpperBoundIsWithinTheLimit(t *testing.T) {
	obs := manyLadders(30, 0.80, []float64{0.60, 0.40, 0.10}, 0.01)
	one := ladderObservations(0.80, []float64{0.60, 0.95, 0.10}, 0.01) // one parent with a rise beyond the noise
	for i := range one {
		one[i].CaseID += "-bad"
		if one[i].ParentID != "" {
			one[i].ParentID += "-bad"
		}
	}
	gate := gateOf(t, append(obs, one...), DefaultLimits())
	assert.Equal(t, GatePassed, gate.Status, "%+v", gate)
	assert.Contains(t, gate.Detail, "of 93 steps inverted")
	assert.LessOrEqual(t, gate.Value, DefaultLimits().MaxInversionRate)
}

func TestUnderGates2ManyInvertedStepsFail(t *testing.T) {
	obs := manyLadders(30, 0.80, []float64{0.20, 0.70, 0.10}, 0.01)
	gate := gateOf(t, obs, DefaultLimits())
	assert.Equal(t, GateFailed, gate.Status)
	assert.Contains(t, gate.Detail, "inverted in strip_task_types")
}

func TestUnderGates2TooFewStepsAreNotMeasuredRatherThanFailed(t *testing.T) {
	gate := gateOf(t, ladderObservations(0.80, []float64{0.60, 0.40, 0.10}, 0.01), DefaultLimits())
	assert.Equal(t, GateNotMeasured, gate.Status)
	assert.Contains(t, gate.Detail, "needs at least")
}

func TestARecordMadeBeforeTheNoiseTermStaysValid(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), DefaultConfig())
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: "handoff-v1", ProfileVersion: "abcd1234", Now: fixedNow})
	require.NoError(t, err)
	record.Limits.NoiseFactor = 0
	sealed, err := record.digest()
	require.NoError(t, err)
	record.Integrity = sealed
	record.ID = "cal-20261004-" + sealed[len("sha256:"):][:8]
	require.NoError(t, record.Validate(), "a zero noise factor is omitted from the JSON, so the old digest is unchanged")
}

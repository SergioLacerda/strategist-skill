package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func feasibleRows(feasible ...bool) []SweepRow {
	rows := make([]SweepRow, len(feasible))
	for i, f := range feasible {
		rows[i] = SweepRow{Threshold: 0.5 + float64(i)/100, Feasible: f}
	}
	return rows
}

func TestWidestBandFindsTheLongestRun(t *testing.T) {
	start, length := widestBand(feasibleRows(false, true, true, false, true, true, true, false))
	assert.Equal(t, 4, start)
	assert.Equal(t, 3, length)

	_, none := widestBand(feasibleRows(false, false))
	assert.Zero(t, none)
}

func TestRecommendCenteredSitsInTheMiddleOfTheBand(t *testing.T) {
	threshold, ok := RecommendCentered(feasibleRows(false, true, true, true, true, true, false))
	require.True(t, ok)
	assert.InDelta(t, 0.53, threshold, 1e-9, "the band is 0.51 to 0.55, its middle is 0.53")

	_, ok = RecommendCentered(feasibleRows(false, false))
	assert.False(t, ok)
}

func noisyStat(scores ...float64) CaseStat {
	stat := CaseStat{ID: "c"}
	for _, score := range scores {
		stat.Observations = append(stat.Observations, Observation{PAccept: score})
	}
	return stat
}

func TestBandGateComparesTheBandWithTheNoise(t *testing.T) {
	limits := Gates3Limits()
	narrow := bandGate(feasibleRows(true, true), []CaseStat{noisyStat(0.5, 0.7)}, limits)
	wide := bandGate(feasibleRows(true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true), []CaseStat{noisyStat(0.5, 0.7)}, limits)
	unmeasured := bandGate(feasibleRows(true), []CaseStat{noisyStat(0.5)}, limits)

	assert.Equal(t, GateFailed, narrow.Status)
	assert.Equal(t, GatePassed, wide.Status)
	assert.Equal(t, GateNotMeasured, unmeasured.Status, "without repetitions the noise is unknown")
}

func TestFoldKeepsAPackageAndItsMutantsTogether(t *testing.T) {
	first, again := FoldOf("parent-1", 5), FoldOf("parent-1", 5)
	assert.Equal(t, first, again, "a pure function of the id")
	seen := map[int]bool{}
	for i := range 40 {
		seen[FoldOf(string(rune('a'+i%26))+string(rune('a'+i/26)), 5)] = true
	}
	assert.Len(t, seen, 5, "forty parents reach every one of five folds")

	parent := CaseStat{ID: "p", Kind: KindGolden}
	child := CaseStat{ID: "p~x", ParentID: "p", Kind: KindSynthetic}
	assert.Equal(t, FoldOf(foldKey(parent), 5), FoldOf(foldKey(child), 5))
}

func TestCrossValidationConfirmsOnUpperBounds(t *testing.T) {
	policy := DefaultPolicy()
	good := CrossValidation{Folds: 5, FalseApproval: NewRate(0, 200), FalseRejection: NewRate(0, 30)}
	loose := CrossValidation{Folds: 5, FalseApproval: NewRate(0, 200), FalseRejection: NewRate(1, 7)}
	gap := CrossValidation{Folds: 5, Infeasible: 1, FalseApproval: NewRate(0, 200), FalseRejection: NewRate(0, 30)}

	assert.True(t, good.Confirmed(policy))
	assert.False(t, loose.Confirmed(policy), "one in seven is a point estimate under the limit and an upper bound far above it")
	assert.False(t, gap.Confirmed(policy), "a fold with no feasible threshold cannot confirm")
	assert.Contains(t, loose.Note(policy), "false rejection upper bound")
}

func TestGates3RecommendsACenteredThresholdForAFitJudge(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), ConfigForRule(RuleGates3))

	require.Equal(t, StatusRecommended, report.Status, "%+v %v", report.Gates, report.Notes)
	require.NotNil(t, report.CrossValidation)
	assert.Nil(t, report.HeldOut, "the single held-out split is not the confirmation under gates-3")
	assert.Equal(t, 5, report.CrossValidation.Folds)
	assert.Zero(t, report.CrossValidation.Infeasible)
	assert.Equal(t, RuleGates3, report.Limits.Rule)
	names := map[string]bool{}
	for _, gate := range report.Gates {
		names[gate.Name] = gate.Status == GatePassed
	}
	assert.True(t, names[GateThresholdBand])
	assert.Greater(t, *report.Threshold, 0.5)
}

func TestGates3KeepsGatesAndSampleRulesBeforeTheConfirmation(t *testing.T) {
	flat := FuncJudge(func(string, Fields) Verdict { return ScoredVerdict(0.9) })
	notFit := Evaluate(runAll(t, corpus(40), flat, 2), ConfigForRule(RuleGates3))
	assert.Equal(t, StatusNotFit, notFit.Status)
	assert.Equal(t, GateSeparation, notFit.FailingGate)

	small := Evaluate(runAll(t, corpus(6), FuncJudge(ruleJudge), 2), ConfigForRule(RuleGates3))
	assert.Equal(t, StatusInsufficientSample, small.Status)
	assert.Nil(t, small.CrossValidation)
}

func TestGates2StillUsesTheHeldOutSplit(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), ConfigForRule(RuleGates2))
	require.Equal(t, StatusRecommended, report.Status)
	assert.NotNil(t, report.HeldOut)
	assert.Nil(t, report.CrossValidation)
}

func TestARecordCarriesTheCrossValidation(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), ConfigForRule(RuleGates3))
	record, err := NewRecord(report, Meta{Provider: "jev", Model: "m", Profile: ProfileV2, ProfileVersion: "v", Now: fixedNow})
	require.NoError(t, err)
	require.NotNil(t, record.CrossValidation)
	assert.NoError(t, record.Validate())
}

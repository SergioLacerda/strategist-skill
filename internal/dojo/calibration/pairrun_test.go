package calibration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oracleJudge scores every dimension from the pair's own oracle: high where the
// relation holds, low where it does not, except for the dimensions in blind, which
// always score the same whatever the pair.
func oracleJudge(blind ...string) FuncPairJudge {
	skip := map[string]bool{}
	for _, d := range blind {
		skip[d] = true
	}
	return func(p Pair) PairVerdict {
		scores := map[string]float64{}
		for _, dim := range PairDimensions() {
			switch {
			case skip[dim]:
				scores[dim] = 0.6
			case p.Oracle[dim]:
				scores[dim] = 0.9
			default:
				scores[dim] = 0.1
			}
		}
		return PairVerdict{Scores: scores, Model: "scripted-1", InputTokens: 100, OutputTokens: 10}
	}
}

func runPairs(t *testing.T, judge PairJudge, n, repetitions int) []PairObservation {
	t.Helper()
	var out []PairObservation
	summary, err := RunPairs(context.Background(), SyntheticPairs(n), judge, PairRunOptions{
		Repetitions: repetitions, Profile: "atomic-v1", ProfileVersion: "abcd1234", Session: "s1",
		HeldOut: func(parent string) bool { return PairHeldOut(parent, "salt", 30) },
		Sink:    func(o PairObservation) error { out = append(out, o); return nil },
	})
	require.NoError(t, err)
	assert.Equal(t, n*6*repetitions, summary.Calls)
	assert.Equal(t, summary.InputTokens, 100*(summary.Calls-summary.Failures))
	return out
}

func statusOf(report PairReport, dim string) DimensionReport {
	for _, d := range report.Dimensions {
		if d.Dimension == dim {
			return d
		}
	}
	return DimensionReport{}
}

func TestAJudgeThatSeesEveryRelationIsFitOnEveryDimension(t *testing.T) {
	report := EvaluatePairs(runPairs(t, oracleJudge(), 12, 2), DefaultPairLimits())
	for _, dim := range PairDimensions() {
		d := statusOf(report, dim)
		assert.Equal(t, DimensionFit, d.Status, "%s %s", dim, d.Describe())
		assert.InDelta(t, 1.0, d.AUC, 1e-9)
		assert.Equal(t, 12, d.Parents)
	}
}

func TestAJudgeBlindToOneDimensionIsNotFitThereAndOnlyThere(t *testing.T) {
	report := EvaluatePairs(runPairs(t, oracleJudge(DimObligation), 12, 2), DefaultPairLimits())
	d := statusOf(report, DimObligation)
	assert.Equal(t, DimensionNotFit, d.Status)
	assert.Equal(t, GateSeparation, d.FailingGate)
	assert.Equal(t, DimensionFit, statusOf(report, DimKind).Status)
	assert.Equal(t, DimensionFit, statusOf(report, DimVerification).Status)
}

func TestTooFewParentsAreInsufficientNotFit(t *testing.T) {
	report := EvaluatePairs(runPairs(t, oracleJudge(), 4, 2), DefaultPairLimits())
	assert.Equal(t, DimensionInsufficientSample, statusOf(report, DimKind).Status)
}

func TestAnUnstableJudgeFailsTheStabilityGate(t *testing.T) {
	calls := 0
	noisy := FuncPairJudge(func(p Pair) PairVerdict {
		calls++
		v := oracleJudge()(p)
		jitter := 0.25
		if calls%5 < 2 {
			jitter = -0.25
		}
		for dim := range v.Scores {
			v.Scores[dim] = min(max(v.Scores[dim]+jitter*0.8, 0), 1)
		}
		return v
	})
	report := EvaluatePairs(runPairs(t, noisy, 12, 2), DefaultPairLimits())
	d := statusOf(report, DimKind)
	assert.Equal(t, DimensionNotFit, d.Status)
	assert.Equal(t, GateStability, d.FailingGate)
}

func TestFamiliesAreReadAgainstTheirFaithfulSibling(t *testing.T) {
	report := EvaluatePairs(runPairs(t, oracleJudge(), 12, 2), DefaultPairLimits())
	byFamily := map[string]FamilyReport{}
	for _, f := range report.Families {
		byFamily[f.Family] = f
	}
	for _, mutant := range []string{FamilyWeakened, FamilyWrongKind, FamilyWrongCheck} {
		assert.Equal(t, 12, byFamily[mutant].Responded.K, mutant)
		assert.NotEmpty(t, byFamily[mutant].Dimension)
	}
	for _, control := range []string{FamilyParaphrase, FamilyDistractor} {
		assert.Equal(t, 12, byFamily[control].Responded.K, "%s stays within the epsilon", control)
		assert.Empty(t, byFamily[control].Dimension)
	}
}

func TestADistractorThatMovesTheScoreIsCaught(t *testing.T) {
	judge := FuncPairJudge(func(p Pair) PairVerdict {
		v := oracleJudge()(p)
		if p.Family == FamilyDistractor {
			v.Scores[DimObligation] = 0.4 // the planted sentence changed the verdict
		}
		return v
	})
	report := EvaluatePairs(runPairs(t, judge, 12, 2), DefaultPairLimits())
	for _, f := range report.Families {
		if f.Family == FamilyDistractor {
			assert.Zero(t, f.Responded.K, "no distractor pair stayed within the epsilon")
			assert.Greater(t, f.MeanDelta, PairEpsilon)
		}
	}
}

func TestFailedCallsAreCountedAndKeptOutOfTheRates(t *testing.T) {
	failing := FuncPairJudge(func(Pair) PairVerdict { return PairVerdict{State: "timeout"} })
	obs := runPairs(t, failing, 3, 1)
	report := EvaluatePairs(obs, DefaultPairLimits())
	assert.Equal(t, len(obs), report.Failed)
	assert.Equal(t, DimensionInsufficientSample, statusOf(report, DimKind).Status)
}

func TestPairObservationsNeverCarryTheText(t *testing.T) {
	for _, o := range runPairs(t, oracleJudge(), 2, 1) {
		raw, err := jsonOf(o)
		require.NoError(t, err)
		assert.NotContains(t, raw, "consumers")
		assert.NotContains(t, raw, "statement")
	}
}

func TestPairRunStopsAtTheBudgetAndOnACancelledContext(t *testing.T) {
	summary, err := RunPairs(context.Background(), SyntheticPairs(3), oracleJudge(), PairRunOptions{Repetitions: 2, Budget: 5})
	require.NoError(t, err)
	assert.Equal(t, 5, summary.Calls)
	assert.Equal(t, 3*6*2-5, summary.Skipped)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RunPairs(ctx, SyntheticPairs(1), oracleJudge(), PairRunOptions{})
	require.Error(t, err)
}

func jsonOf(o PairObservation) (string, error) {
	raw, err := json.Marshal(o)
	return string(raw), err
}

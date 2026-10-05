package calibration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func corpus(n int) []Source {
	sources := make([]Source, n)
	for i := range sources {
		fields := packageFields(i)
		digest := Digest(fields)
		sources[i] = Source{Name: "pkg", Digest: digest, Boundary: BoundaryPackage, Fields: fields, Golden: true, HeldOut: HeldOut(digest, "test", 25)}
	}
	return sources
}

func runAll(t *testing.T, sources []Source, judge Judge, repetitions int) []Observation {
	t.Helper()
	var cases []Case
	for _, src := range sources {
		cases = append(cases, Generate(src, Options{Seed: 1})...)
	}
	var out []Observation
	_, err := Run(context.Background(), cases, judge, RunOptions{Repetitions: repetitions, Profile: "handoff-v1", Sink: func(o Observation) error { out = append(out, o); return nil }})
	require.NoError(t, err)
	return out
}

func TestEvaluateRecommendsAThresholdForAFitJudge(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 2), DefaultConfig())

	require.Equal(t, StatusRecommended, report.Status, "%+v %v", report.Gates, report.Notes)
	require.NotNil(t, report.Threshold)
	assert.Greater(t, *report.Threshold, 0.5)
	assert.InDelta(t, 0.0, report.HeldOut.FalseApproval.Value, 1e-9)
	assert.InDelta(t, 0.0, report.HeldOut.FalseRejection.Value, 1e-9)
	assert.InDelta(t, 1.0, report.AUC, 1e-9)
	for _, gate := range report.Gates {
		assert.Equal(t, GatePassed, gate.Status, gate.Name)
	}
	assert.InDelta(t, 0.0, report.AtCurrent.FalseApproval.Value, 1e-9)
}

func TestEvaluateRefusesAJudgeThatDoesNotSeparate(t *testing.T) {
	flat := FuncJudge(func(string, Fields) Verdict { return ScoredVerdict(0.9) })
	report := Evaluate(runAll(t, corpus(40), flat, 2), DefaultConfig())

	assert.Equal(t, StatusNotFit, report.Status)
	assert.Equal(t, GateSeparation, report.FailingGate)
	assert.Nil(t, report.Threshold)
	assert.InDelta(t, 1.0, report.AtCurrent.FalseApproval.Value, 1e-9, "the default threshold would approve every defect")
}

func TestEvaluateNeedsRepetitionsForStability(t *testing.T) {
	report := Evaluate(runAll(t, corpus(120), FuncJudge(ruleJudge), 1), DefaultConfig())
	assert.Equal(t, StatusNotFit, report.Status)
	assert.Equal(t, GateStability, report.FailingGate)
}

func TestEvaluateReportsInsufficientSampleInsteadOfCertifying(t *testing.T) {
	report := Evaluate(runAll(t, corpus(8), FuncJudge(ruleJudge), 2), DefaultConfig())
	assert.Equal(t, StatusInsufficientSample, report.Status)
	assert.NotEmpty(t, report.Notes)
	assert.Nil(t, report.Threshold)
}

func TestInsensitiveOperatorsAreNamed(t *testing.T) {
	blind := FuncJudge(func(_ string, f Fields) Verdict {
		if f["approved_scope"] == "" {
			return ScoredVerdict(0.1)
		}
		return ScoredVerdict(0.95)
	})
	report := Evaluate(runAll(t, corpus(40), blind, 2), DefaultConfig())
	insensitive := map[string]bool{}
	for _, op := range report.Operators {
		if op.Insensitive() {
			insensitive[op.Operator] = true
		}
	}
	assert.True(t, insensitive["weaken_must_to_should"])
	assert.True(t, insensitive["strip_task_types"])
	assert.False(t, insensitive["drop_approved_scope"])
}

func TestNeutralPerturbationsAreMeasuredAsInvariance(t *testing.T) {
	report := Evaluate(runAll(t, corpus(40), FuncJudge(ruleJudge), 2), DefaultConfig())
	for _, op := range report.Operators {
		if op.Neutral {
			assert.InDelta(t, 1.0, op.Sensitivity.Value, 1e-9, op.Operator)
		}
	}
}

func TestAnUnauditedStage2CaseNeverCounts(t *testing.T) {
	var cases []Case
	for _, src := range corpus(10) {
		cases = append(cases, Generate(src, Options{Seed: 1, MaxStage: 2, Others: corpus(3)})...)
	}
	var obs []Observation
	_, err := Run(context.Background(), cases, FuncJudge(ruleJudge), RunOptions{Sink: func(o Observation) error { obs = append(obs, o); return nil }})
	require.NoError(t, err)
	without := Stats(obs, Filter{})
	for _, stat := range without {
		assert.Less(t, stat.Stage, 2)
	}
	audited := map[string]bool{}
	for _, c := range cases {
		if c.Stage == 2 {
			audited[c.ID] = true
		}
	}
	with := Stats(obs, Filter{Audited: audited})
	assert.Greater(t, len(with), len(without))
	excluded := map[string]bool{}
	for id := range audited {
		excluded[id] = true
	}
	assert.Len(t, Stats(obs, Filter{Audited: audited, Excluded: excluded}), len(without))
}

func TestRunStopsAtTheBudgetAndRecordsNoContent(t *testing.T) {
	cases := Generate(source(1), Options{Seed: 1})
	var obs []Observation
	summary, err := Run(context.Background(), cases, FuncJudge(ruleJudge), RunOptions{Repetitions: 2, Budget: 5, Sink: func(o Observation) error { obs = append(obs, o); return nil }})
	require.NoError(t, err)
	assert.Equal(t, 5, summary.Calls)
	assert.Equal(t, len(cases)*2-5, summary.Skipped)
	assert.Equal(t, Plan(cases, 2), len(cases)*2)
	raw, err := obs[0].Encode()
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "README")
	decoded, bad := DecodeObservations(append(raw, []byte("\nnot json\n")...))
	assert.Len(t, decoded, 1)
	assert.Equal(t, 1, bad)
}

func TestRunCountsJudgeFailuresAndKeepsThemOutOfTheRates(t *testing.T) {
	failing := FuncJudge(func(string, Fields) Verdict { return Verdict{State: "timeout"} })
	var obs []Observation
	summary, err := Run(context.Background(), Generate(source(1), Options{Seed: 1}), failing, RunOptions{Sink: func(o Observation) error { obs = append(obs, o); return nil }})
	require.NoError(t, err)
	assert.Equal(t, summary.Calls, summary.Failures)
	report := Evaluate(obs, DefaultConfig())
	assert.Equal(t, 0, report.Counts.Golden)
	assert.Equal(t, summary.Failures, report.Counts.Failed)
}

func TestRunStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Run(ctx, Generate(source(1), Options{Seed: 1}), FuncJudge(ruleJudge), RunOptions{})
	assert.Error(t, err)
}

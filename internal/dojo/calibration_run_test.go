package dojo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runTime = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func runInput(t *testing.T, w workspace, judge calibration.Judge, mutate func(*domain.DojoCalibration)) dojo.RunInput {
	t.Helper()
	cal := policyCase()
	cal.Corpus.HeldOutPercent = 25
	if mutate != nil {
		mutate(&cal)
	}
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	paths, err := dojo.NewCasePaths(w.base, "jev-handoff", cal.Output)
	require.NoError(t, err)
	return dojo.RunInput{
		BasePath: w.base, Criteria: criteriaFor("jev-handoff", cal), Paths: paths, Corpus: corpus, Judge: judge,
		Profile: "handoff-v1", ProfileVersion: "v00000001", Repetitions: dojo.Repetitions(cal), Current: 0.80, Now: runTime,
	}
}

func goldenWorkspace(t *testing.T, n int) workspace {
	t.Helper()
	w := newWorkspace(t)
	w.packages(t, n)
	cal := policyCase()
	cal.Corpus.HeldOutPercent = 25
	w.promoteAll(t, cal)
	return w
}

func TestRunCalibrationProducesEvidenceObservationsAndARecommendedRecord(t *testing.T) {
	w := goldenWorkspace(t, 120)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)

	out, err := dojo.RunCalibration(context.Background(), in)

	require.NoError(t, err)
	require.NotNil(t, out.Record)
	assert.Equal(t, calibration.StatusRecommended, out.Record.Status, "%v", out.Check.Report.Gates)
	assert.True(t, out.Check.Result.Passed(), "%+v", out.Check.Result.Items)
	require.NotNil(t, out.Check.Result.Calibration)
	assert.Equal(t, out.Record.ID, out.Check.Result.Calibration.RecordID)

	loaded, err := dojo.LoadCalibration(w.base, out.Record.ID)
	require.NoError(t, err)
	assert.Equal(t, out.Record.Integrity, loaded.Integrity)
	assert.FileExists(t, in.Paths.ObservationsPath)
	assert.FileExists(t, in.Paths.CorpusPath)
	observed, bad, err := dojo.ReadObservations(in.Paths.ObservationsPath)
	require.NoError(t, err)
	assert.Zero(t, bad)
	assert.Len(t, observed, out.Summary.Calls)
}

func TestObservationsNeverCarryPackageContent(t *testing.T) {
	w := goldenWorkspace(t, 6)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	_, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)
	raw, err := os.ReadFile(in.Paths.ObservationsPath)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "README")
	assert.NotContains(t, string(raw), "internal/dojo")
}

func TestCheckRecomputesFromObservationsWithNoCalls(t *testing.T) {
	w := goldenWorkspace(t, 120)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	ran, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)

	checked, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: 0.80, ProfileVersion: "v00000001", Session: "latest"})

	require.NoError(t, err)
	assert.Equal(t, ran.Check.Report.Status, checked.Report.Status)
	assert.Equal(t, ran.Check.Report.Threshold, checked.Report.Threshold)
	assert.True(t, checked.Result.Passed())
}

func TestCheckFailsWithoutObservationsAndNamesWhy(t *testing.T) {
	w := newWorkspace(t)
	paths, err := dojo.NewCasePaths(w.base, "empty", "")
	require.NoError(t, err)
	checked, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: criteriaFor("empty", policyCase()), Paths: paths, Current: 0.8})
	require.NoError(t, err)
	assert.False(t, checked.Result.Passed())
	assert.Equal(t, []dojo.FailureReason{dojo.FailureNotCalibrated}, dojo.ClassifyFailures(checked.Result.Items))
}

func TestAJudgeThatApprovesEverythingFailsWithFalseApprovalAndNoRecommendation(t *testing.T) {
	w := goldenWorkspace(t, 40)
	flat := calibration.FuncJudge(func(string, calibration.Fields) calibration.Verdict { return calibration.ScoredVerdict(0.93) })
	in := runInput(t, w, flat, nil)
	out, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, calibration.StatusNotFit, out.Check.Report.Status)
	assert.Nil(t, out.Record.Threshold)
	reasons := dojo.ClassifyFailures(out.Check.Result.Items)
	assert.Contains(t, reasons, dojo.FailureFalseApproval)
	assert.Contains(t, reasons, dojo.FailureNotSeparating)
	lesson := dojo.GenerateCalibrationLesson(out.Check.Result, out.Check.Report, nil)
	assert.Contains(t, lesson, "Weak Spots")
	assert.Contains(t, lesson, "strip_task_types")
}

func TestARunStopsAtTheCallBudget(t *testing.T) {
	w := goldenWorkspace(t, 10)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	in.Budget = 25
	out, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, 25, out.Summary.Calls)
	assert.Positive(t, out.Summary.Skipped)
}

func TestBoundaryCasesAreRepeatedWithinTheSession(t *testing.T) {
	w := goldenWorkspace(t, 10)
	near := calibration.FuncJudge(func(_ string, f calibration.Fields) calibration.Verdict {
		if f["approved_scope"] == "" {
			return calibration.ScoredVerdict(0.78)
		}
		return calibration.ScoredVerdict(0.95)
	})
	in := runInput(t, w, near, nil)
	in.BoundaryMargin = 0.05
	out, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)
	observed, _, err := dojo.ReadObservations(in.Paths.ObservationsPath)
	require.NoError(t, err)
	highest := 0
	for _, o := range observed {
		highest = max(highest, o.Run)
		assert.Equal(t, out.Session, o.Session)
	}
	assert.Greater(t, highest, in.Repetitions, "near-threshold cases got extra repetitions")
}

func TestAnAllFailingJudgeLeavesNoRecordAndSaysWhy(t *testing.T) {
	w := goldenWorkspace(t, 5)
	failing := calibration.FuncJudge(func(string, calibration.Fields) calibration.Verdict { return calibration.Verdict{State: "timeout"} })
	_, err := dojo.RunCalibration(context.Background(), runInput(t, w, failing, nil))
	require.Error(t, err)
	ids, listErr := dojo.ListCalibrations(w.base)
	require.NoError(t, listErr)
	assert.Empty(t, ids)
}

func TestAConfiguredOutputPathHoldsTheCaseState(t *testing.T) {
	w := goldenWorkspace(t, 6)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), func(c *domain.DojoCalibration) { c.Output = "dojo/outputs/jev" })
	_, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(w.base, "dojo", "outputs", "jev", "observations.jsonl"))
	assert.FileExists(t, filepath.Join(w.base, "dojo", "outputs", "jev", "corpus.json"))
}

func TestACorpusWithoutCasesIsRefused(t *testing.T) {
	w := newWorkspace(t)
	_, err := dojo.RunCalibration(context.Background(), runInput(t, w, calibration.FuncJudge(ruleJudge), nil))
	require.Error(t, err)
}

func TestTheStaleProfileIsReportedWhenTheQuestionsChange(t *testing.T) {
	w := goldenWorkspace(t, 6)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	_, err := dojo.RunCalibration(context.Background(), in)
	require.NoError(t, err)
	checked, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: 0.8, ProfileVersion: "changed00", Session: "latest"})
	require.NoError(t, err)
	assert.Contains(t, dojo.ClassifyFailures(checked.Result.Items), dojo.FailureStaleCalibration)
}

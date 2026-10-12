package dojo_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func someRecord(t *testing.T) calibration.Record {
	t.Helper()
	record, err := calibration.NewRecord(calibration.Report{Status: calibration.StatusNotFit, FailingGate: "separation"},
		calibration.Meta{Provider: "jev", Model: "jev-1.13.0", Profile: "handoff-v1", ProfileVersion: "abcd1234", Now: runTime})
	require.NoError(t, err)
	return record
}

func TestSaveAndLoadCalibration(t *testing.T) {
	base := t.TempDir()
	record := someRecord(t)
	require.NoError(t, dojo.SaveCalibration(base, record))
	require.Error(t, dojo.SaveCalibration(base, record), "a published id is never overwritten")

	loaded, err := dojo.LoadCalibration(base, record.ID)
	require.NoError(t, err)
	assert.Equal(t, record.ID, loaded.ID)
	ids, err := dojo.ListCalibrations(base)
	require.NoError(t, err)
	assert.Equal(t, []string{record.ID}, ids)
}

func TestLoadCalibrationRefusesNonIDsAndTamperedFiles(t *testing.T) {
	base := t.TempDir()
	for _, bad := range []string{"latest", "../../etc/passwd", "cal-1", ""} {
		_, err := dojo.LoadCalibration(base, bad)
		require.Error(t, err, bad)
	}
	record := someRecord(t)
	require.NoError(t, dojo.SaveCalibration(base, record))
	path := filepath.Join(base, "dojo", dojo.DirCalibrations, record.ID+".json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var edited map[string]any
	require.NoError(t, json.Unmarshal(raw, &edited))
	edited["model"] = "another-model"
	raw, err = json.Marshal(edited)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	_, err = dojo.LoadCalibration(base, record.ID)
	require.Error(t, err)
}

func TestSaveCalibrationRefusesAnInvalidRecord(t *testing.T) {
	record := someRecord(t)
	record.Model = ""
	require.Error(t, dojo.SaveCalibration(t.TempDir(), record))
}

func TestAuditSampleIsDeterministicAndStartsWithStage2(t *testing.T) {
	w := newWorkspace(t)
	w.packages(t, 3)
	cal := policyCase()
	cal.Stage = 2
	corpus, err := dojo.BuildCorpus(w.base, cal, stubReader{})
	require.NoError(t, err)
	cases := dojo.BuildCases(corpus, cal)
	first := dojo.SampleForAudit(cases, 6, 1)
	second := dojo.SampleForAudit(cases, 6, 1)
	require.Len(t, first, 6)
	for i := range first {
		assert.Equal(t, first[i].ID, second[i].ID)
	}
	assert.Equal(t, 2, first[0].Stage)
	assert.Len(t, dojo.SampleForAudit(cases, 10000, 1), len(dojo.SampleForAudit(cases, 10000, 2)))
	assert.Empty(t, dojo.SampleForAudit(nil, 3, 1))
}

func TestAuditEntriesDriveTheMetricsFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	require.Error(t, dojo.RecordAudit(path, dojo.AuditEntry{CaseID: "c", Verdict: "agree"}, runTime), "a reviewer is required")
	require.Error(t, dojo.RecordAudit(path, dojo.AuditEntry{CaseID: "c", Verdict: "maybe", By: "x"}, runTime))
	require.NoError(t, dojo.RecordAudit(path, dojo.AuditEntry{CaseID: "a", Verdict: dojo.VerdictAgree, By: "r"}, runTime))
	require.NoError(t, dojo.RecordAudit(path, dojo.AuditEntry{CaseID: "b", Verdict: dojo.VerdictDisagree, By: "r", Note: "not a real defect"}, runTime))
	require.NoError(t, dojo.RecordAudit(path, dojo.AuditEntry{CaseID: "a", Verdict: dojo.VerdictDisagree, By: "r", Note: "changed my mind"}, runTime))

	filter, entries, err := dojo.LoadAudit(path)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.True(t, filter.Excluded["a"] && filter.Excluded["b"], "the last entry wins")
	assert.False(t, filter.Audited["a"])
}

func TestAuditDisagreementsLeaveTheMetricsAndAppearInTheLesson(t *testing.T) {
	w := goldenWorkspace(t, 6)
	cal := policyCase()
	cal.Stage = 2
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), func(c *domain.DojoCalibration) { c.Stage = 2 })
	_, err := dojo.RunCalibration(t.Context(), in)
	require.NoError(t, err)
	before, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: 0.8, Session: "latest"})
	require.NoError(t, err)
	require.NoError(t, dojo.RecordAudit(in.Paths.AuditPath, dojo.AuditEntry{CaseID: before.Observations[0].CaseID, Verdict: dojo.VerdictDisagree, By: "reviewer", Note: "mutation does not break the criterion"}, runTime))

	after, err := dojo.CheckCalibration(dojo.CheckInput{Criteria: in.Criteria, Paths: in.Paths, Current: 0.8, Session: "latest"})
	require.NoError(t, err)
	assert.Len(t, after.Audits, 1)
	assert.Contains(t, dojo.GenerateCalibrationLesson(after.Result, after.Report, after.Audits), "mutation does not break the criterion")
}

func TestProposalsAreRecordedPendingAndNeverAddOperators(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, dojo.AppendProposal(base, dojo.Proposal{Name: "swap_sections", Description: "swap two sections", By: "agent"}, runTime))
	require.Error(t, dojo.AppendProposal(base, dojo.Proposal{Name: "drop_approved_scope", Description: "dup"}, runTime))
	require.Error(t, dojo.AppendProposal(base, dojo.Proposal{Name: "../x", Description: "bad"}, runTime))
	require.Error(t, dojo.AppendProposal(base, dojo.Proposal{Name: "no_description"}, runTime))
	raw, err := os.ReadFile(filepath.Join(base, "dojo", dojo.DirMutations, "proposals.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"status":"pending"`)
	_, known := calibration.ByName("swap_sections")
	assert.False(t, known)
}

func TestCriteriaValidationOfACalibrationCase(t *testing.T) {
	valid := criteriaFor("case", policyCase())
	require.NoError(t, dojo.ValidateCriteria(valid))

	for name, mutate := range map[string]func(*domain.DojoCriteria){
		"no block":         func(c *domain.DojoCriteria) { c.Calibration = nil },
		"unknown kind":     func(c *domain.DojoCriteria) { c.Kind = "other" },
		"bad boundary":     func(c *domain.DojoCriteria) { c.Calibration.Boundary = "x" },
		"no risk policy":   func(c *domain.DojoCriteria) { c.Calibration.RiskPolicy = domain.DojoRiskPolicy{} },
		"policy over one":  func(c *domain.DojoCriteria) { c.Calibration.RiskPolicy.MaxFalseApproval = 1.5 },
		"unknown operator": func(c *domain.DojoCriteria) { c.Calibration.Operators = []string{"nope"} },
		"stage three":      func(c *domain.DojoCriteria) { c.Calibration.Stage = 3 },
		"escaping output":  func(c *domain.DojoCriteria) { c.Calibration.Output = "../outside" },
		"escaping root":    func(c *domain.DojoCriteria) { c.Calibration.Corpus.Roots = []string{"/abs"} },
		"held-out too big": func(c *domain.DojoCriteria) { c.Calibration.Corpus.HeldOutPercent = 95 },
		"bad include":      func(c *domain.DojoCriteria) { c.Calibration.Corpus.Include = []string{"a/b"} },
		"bad provider":     func(c *domain.DojoCriteria) { c.Calibration.Provider = "other" },
		"bad layer":        func(c *domain.DojoCriteria) { c.Calibration.Layer = "x" },
	} {
		criteria := criteriaFor("case", policyCase())
		mutate(&criteria)
		require.Error(t, dojo.ValidateCriteria(criteria), name)
	}
	pipeline := domain.DojoCriteria{Scenario: "s", RunDir: "dojo/run", Calibration: &domain.DojoCalibration{}}
	require.Error(t, dojo.ValidateCriteria(pipeline), "a calibration block needs the kind")
	assert.NoError(t, dojo.ValidateCriteria(domain.DojoCriteria{Scenario: "s", RunDir: "dojo/run"}), "a pipeline scenario is unchanged")
}

func TestCalibrationDefaultsResolve(t *testing.T) {
	assert.Equal(t, dojo.DefaultRepetitions, dojo.Repetitions(domain.DojoCalibration{}))
	assert.Equal(t, 5, dojo.Repetitions(domain.DojoCalibration{Repetitions: 5}))
	assert.Equal(t, dojo.DefaultHeldOutPercent, dojo.HeldOutPercent(domain.DojoCalibration{}))
	assert.Equal(t, dojo.DefaultMaxAgeDays, dojo.MaxAgeDays(domain.DojoCalibration{}))
}

func TestPersistResultCarriesTheCalibrationSection(t *testing.T) {
	base := t.TempDir()
	threshold := 0.84
	result := domain.DojoCheckResult{Scenario: "jev-handoff", Items: []domain.DojoCheckItem{{Label: "calibration status", Passed: true}},
		Calibration: &domain.DojoCalibrationResult{Status: "recommended", Threshold: &threshold, RecordID: "cal-20261004-aaaaaaaa"}}
	require.NoError(t, dojo.PersistResult(base, result, runTime, runTime.Add(time.Second)))
	raw, err := os.ReadFile(filepath.Join(base, "dojo", ".last-run", "jev-handoff", "result.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"calibration"`)
	history, err := os.ReadFile(filepath.Join(base, "dojo", ".history.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(history), "cal-20261004-aaaaaaaa")
	require.NoError(t, dojo.WriteLesson(base, result))
	assert.NoFileExists(t, filepath.Join(base, "dojo", ".last-run", "jev-handoff", "lesson.md"), "a passing run writes no lesson")
}

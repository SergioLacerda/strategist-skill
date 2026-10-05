package dojo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCandidatesFindsOnlyDisagreements(t *testing.T) {
	signals := []dojo.MissionSignal{
		{Transition: "archivist_to_gate", Answered: true, Approved: false, Confidence: 0.26, Threshold: 0.8, GateOutcome: "accepted"},
		{Transition: "ranger_to_archivist", Answered: true, Approved: true, Confidence: 0.9, Threshold: 0.8, GateOutcome: "revision_requested"},
		{Transition: "archivist_to_gate", Answered: true, Approved: true, GateOutcome: "accepted"},
		{Transition: "archivist_to_gate", Answered: false, GateOutcome: "accepted"},
		{Transition: "archivist_to_gate", Answered: true, Approved: false, GateOutcome: ""},
	}
	found := dojo.DetectCandidates("m1", signals, runTime)
	require.Len(t, found, 2)
	assert.Equal(t, dojo.CandidateRejectedAccepted, found[0].Kind)
	assert.Equal(t, dojo.CandidateApprovedRevised, found[1].Kind)
	assert.Equal(t, "m1", found[0].Package)
}

func TestEvidenceAndManifestDigestRoundTrip(t *testing.T) {
	w := goldenWorkspace(t, 6)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), nil)
	out, err := dojo.RunCalibration(t.Context(), in)
	require.NoError(t, err)

	evidence := dojo.NewEvidence("jev-handoff", "judge", out.Check, dojo.ReadManifestDigest(in.Paths), out.Summary, runTime)
	require.NoError(t, dojo.WriteEvidence(in.Paths, evidence))
	assert.FileExists(t, in.Paths.EvidencePath)
	assert.Equal(t, in.Corpus.Manifest.Digest, evidence.Corpus)
	assert.Equal(t, dojo.EvidenceSchema, evidence.Schema)
	assert.Equal(t, out.Record.ID, evidence.RecordID)
	assert.Empty(t, dojo.ReadManifestDigest(dojo.CasePaths{CorpusPath: filepath.Join(t.TempDir(), "none.json")}))
	require.NoError(t, os.WriteFile(in.Paths.CorpusPath, []byte("{"), 0o644))
	assert.Empty(t, dojo.ReadManifestDigest(in.Paths))
}

func TestCaseSummariesAndBytes(t *testing.T) {
	w := newWorkspace(t)
	w.packages(t, 3)
	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	cases := dojo.BuildCases(corpus, policyCase())
	counts := dojo.Summary(cases)
	assert.Equal(t, 3, counts[calibration.KindCandidate])
	assert.Equal(t, dojo.CountDefects(cases), countRejects(cases))
	assert.Equal(t, 10*len(cases)*2, dojo.PlannedBytes(cases, 2, func(calibration.Case) int { return 10 }))
	assert.Equal(t, 10*len(cases), dojo.PlannedBytes(cases, 0, func(calibration.Case) int { return 10 }))
}

func countRejects(cases []calibration.Case) int {
	n := 0
	for _, c := range cases {
		if c.Expect == calibration.ExpectReject {
			n++
		}
	}
	return n
}

func TestCalibrationLessonIsWrittenOnlyForAFailedCase(t *testing.T) {
	base := t.TempDir()
	passing := dojo.CheckOutcome{Result: domain.DojoCheckResult{Scenario: "case", Items: []domain.DojoCheckItem{{Label: "calibration status", Passed: true}}}}
	require.NoError(t, dojo.WriteCalibrationLesson(base, passing))
	assert.NoFileExists(t, filepath.Join(base, "dojo", ".last-run", "case", "lesson.md"))

	failing := dojo.CheckOutcome{Result: domain.DojoCheckResult{Scenario: "case", Items: []domain.DojoCheckItem{{Label: "calibration gate separation", Passed: false, Detail: "auc 0.6"}}}}
	require.NoError(t, dojo.WriteCalibrationLesson(base, failing))
	raw, err := os.ReadFile(filepath.Join(base, "dojo", ".last-run", "case", "lesson.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "not_separating")
	assert.Contains(t, string(raw), "Audit Disagreements")
}

func TestRangerCorpusReadsPendingAnalyses(t *testing.T) {
	w := newWorkspace(t)
	pending := filepath.Join(w.base, "pending")
	require.NoError(t, os.MkdirAll(pending, 0o755))
	artifact := "---\nmission_id: m1\n---\n\n## mission_objective\nDecide how the DOJO calibrates the judge.\n\n## known_facts\n- F-01 the README is stale (`README.md`).\n\n## uncertainties\n- U-01 consent.\n\n## recommended_refinement_focus\nSpecify the corpus.\n"
	require.NoError(t, os.WriteFile(filepath.Join(pending, "m1-analysis.md"), []byte(artifact), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pending, "notes.txt"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(pending, "a-dir-analysis.md"), 0o755))
	cal := policyCase()
	cal.Boundary = calibration.BoundaryRanger

	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})

	require.NoError(t, err)
	require.Len(t, corpus.Manifest.Entries, 1)
	assert.Equal(t, "m1", corpus.Manifest.Entries[0].Name)
	cases := dojo.BuildCases(corpus, cal)
	operators := map[string]bool{}
	for _, c := range cases {
		operators[c.Operator] = true
	}
	assert.True(t, operators["vague_objective"] && operators["empty_refinement_focus"])
}

func TestListCalibrationsIgnoresForeignFiles(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "dojo", dojo.DirCalibrations)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "latest.json"), []byte("{}"), 0o644))
	ids, err := dojo.ListCalibrations(base)
	require.NoError(t, err)
	assert.Empty(t, ids)
	ids, err = dojo.ListCalibrations(filepath.Join(base, "missing"))
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestARunKeepsTheRecordsProviderAndAge(t *testing.T) {
	w := goldenWorkspace(t, 6)
	in := runInput(t, w, calibration.FuncJudge(ruleJudge), func(c *domain.DojoCalibration) { c.Provider = "jev" })
	out, err := dojo.RunCalibration(t.Context(), in)
	require.NoError(t, err)
	assert.Equal(t, "jev", out.Record.Provider)
	assert.WithinDuration(t, runTime, mustTime(t, out.Record.CreatedAt), time.Second)
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}

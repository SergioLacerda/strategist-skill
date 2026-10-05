package dojo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const planText = `- [ ] 1.1 [analysis_artifact] Record the answers in design-%[1]d.md; verify D1 is updated.
- [ ] 1.2 [implementation_handoff] Refresh the README %[1]d; verify dojo list and the README agree.
- [ ] 2.1 [implementation_handoff] Build the manifest builder %[1]d; verify tests cover inclusion.
- [ ] 2.2 [implementation_handoff] Build the catalog %[1]d; verify a golden test per operator. The agent MUST NOT relabel a case.
- [ ] 3.1 [implementation_handoff] Build the evaluator %[1]d; verify it against a scripted judge.
- [ ] 3.2 [analysis_artifact] Report the first calibration %[1]d; verify it states measured values only.
`

const (
	// minimumCalibrationCorpusSize is the smallest size found by the policy
	// bisect: 34 is the first size whose exploratory effective defect sample
	// reaches calibration.MinSample(0.05) for this deterministic corpus.
	minimumCalibrationCorpusSize  = 34
	calibrationCorpusSafetyMargin = 2
	calibrationCorpusSize         = minimumCalibrationCorpusSize + calibrationCorpusSafetyMargin
	belowCalibrationCorpusSize    = minimumCalibrationCorpusSize - 1
)

type rig struct {
	root, base string
	calls      int
	judge      calibration.FuncJudge
	signals    []internaldojo.MissionSignal
	policy     []domain.DojoCheckItem
	live       []domain.DojoCheckItem
	pairCalls  int
	liveCalls  int
}

func newRig(t *testing.T, packages int) *rig {
	t.Helper()
	dir := t.TempDir()
	r := &rig{root: filepath.Join(dir, ".strategist"), base: filepath.Join(dir, ".analysis")}
	require.NoError(t, os.MkdirAll(r.root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(r.root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	for i := 0; i < packages; i++ {
		pkg := filepath.Join(r.base, "done", fmt.Sprintf("pkg-%03d", i))
		require.NoError(t, os.MkdirAll(pkg, 0o755))
		files := map[string]string{
			"analysis.md": "---\nmission_status: documentation_applied\n---\n\n## affected_scope\n- allowed: docs/**\n- forbidden: internal/domain/**\n",
			"proposal.md": "x", "design.md": "x", "tasks.md": "---\nacceptance_checks:\n  - \"the checks pass\"\n---\n" + fmt.Sprintf(planText, i),
		}
		for name, content := range files {
			require.NoError(t, os.WriteFile(filepath.Join(pkg, name), []byte(content), 0o644))
		}
	}
	r.judge = func(_ string, f calibration.Fields) calibration.Verdict {
		r.calls++
		return scriptedScore(f)
	}
	return r
}

func scriptedScore(f calibration.Fields) calibration.Verdict {
	score := 0.95
	plan := f["implementation_plan"]
	if strings.TrimSpace(f["approved_scope"]) == "" || strings.Contains(f["approved_scope"], "every path") {
		score -= 0.5
	}
	tasks := strings.Count(plan, "- [")
	typed := strings.Count(plan, "[analysis_artifact]") + strings.Count(plan, "[implementation_handoff]")
	score -= 0.4 * float64(tasks-typed) / float64(max(tasks, 1))
	if strings.Count(strings.ToLower(plan), "verify") < tasks {
		score -= 0.3
	}
	if strings.Count(plan, "[implementation_handoff]") < 4 {
		score -= 0.5
	}
	if !strings.HasSuffix(strings.TrimSpace(plan), ".") {
		score -= 0.5
	}
	if strings.TrimSpace(f["acceptance_checks"]) == "" {
		score -= 0.5
	}
	if !strings.Contains(plan, "MUST") {
		score -= 0.1
	}
	return calibration.ScoredVerdict(max(score, 0.02))
}

// pairJudge scores every dimension from the pair's own oracle.
func (r *rig) pairJudge(p calibration.Pair) calibration.PairVerdict {
	r.pairCalls++
	scores := map[string]float64{}
	for _, dim := range calibration.PairDimensions() {
		scores[dim] = 0.1
		if p.Oracle[dim] {
			scores[dim] = 0.9
		}
	}
	return calibration.PairVerdict{Scores: scores, Model: "scripted-1", InputTokens: 100, OutputTokens: 10}
}

func (r *rig) deps() Dependencies {
	return Dependencies{
		ResolveRoots: func(string) (string, string, error) {
			s, b, err := cliutil.ResolveActiveBasePath(r.root)
			if err != nil {
				return "", "", fmt.Errorf("dojo: %w", err)
			}
			return s, b, nil
		},
		Calibration: CalibrationDependencies{
			Reader:  handoffconsumer.Reader{},
			Judge:   func(string, string) (calibration.Judge, error) { return r.judge, nil },
			Current: func(string) float64 { return 0.80 },
			Profile: func(profile, b string) (string, string) {
				id := calibration.ResolveProfile(profile)
				return id, handoffconsumer.ProfileVersionOf(id, b)
			},
			Plan: func(_, _, boundary string, cases []calibration.Case) (Plan, error) {
				return Plan{Bytes: len(cases) * 1000, Fields: handoffconsumer.Fields(handoffconsumer.TransitionForBoundary(boundary))}, nil
			},
			Signals:     func(string, string) ([]internaldojo.MissionSignal, error) { return r.signals, nil },
			Policy:      func(time.Time) []domain.DojoCheckItem { return r.policy },
			Endpoint:    func(string) string { return "https://provider.example/v1" },
			PairJudge:   func(string) (calibration.PairJudge, error) { return calibration.FuncPairJudge(r.pairJudge), nil },
			PairProfile: func() (string, string) { return handoffconsumer.PairProfileID, handoffconsumer.PairProfileVersion() },
			LivePlan:    func() (int, int) { return 3, 1200 },
			Live: func(context.Context, string) ([]domain.DojoCheckItem, error) {
				r.liveCalls++
				return r.live, nil
			},
		},
	}
}

func (r *rig) run(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := New(r.deps())
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func (r *rig) generate(t *testing.T, name string) {
	t.Helper()
	_, _, err := r.run(t, "generate", name)
	require.NoError(t, err)
}

func (r *rig) promoteAll(t *testing.T, name string, n int) {
	t.Helper()
	var lines []string
	cc, err := loadCase(r.deps(), "", name)
	require.NoError(t, err)
	corpus, err := internaldojo.BuildCorpus(cc.basePath, *cc.criteria.Calibration, handoffconsumer.Reader{})
	require.NoError(t, err)
	for _, e := range corpus.Manifest.Entries {
		raw, err := json.Marshal(internaldojo.Promotion{Digest: e.Digest, Package: e.Name, ReviewedBy: "r", Oracle: "fixture"})
		require.NoError(t, err)
		lines = append(lines, string(raw))
	}
	require.Len(t, lines, n)
	dir := filepath.Join(r.base, "dojo", "corpus")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "golden.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644))
}

func TestGenerateWritesAValidSkeletonAndNeverOverwrites(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	out, _, err := r.run(t, "generate", "jev-handoff", "--package", "pkg-001")
	require.NoError(t, err)
	assert.Contains(t, out, "next: strategist dojo calibrate jev-handoff --dry-run")
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "jev-handoff", "criteria.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "kind: judge_calibration")
	assert.Contains(t, string(raw), "max_false_approval: 0.05")
	assert.Contains(t, string(raw), "pkg-001")

	_, _, err = r.run(t, "generate", "jev-handoff")
	require.Error(t, err)
	_, _, err = r.run(t, "generate", "corpus")
	require.Error(t, err, "a reserved name is refused")
	_, _, err = r.run(t, "generate", "x", "--boundary", "nope")
	require.Error(t, err)
}

func TestListShowsTheCaseAndNeverTheReservedDirectories(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.generate(t, "jev-handoff")
	require.NoError(t, os.MkdirAll(filepath.Join(r.base, "dojo", "corpus"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(r.base, "dojo", "corpus", "criteria.yaml"), []byte("scenario: corpus\n"), 0o644))
	out, _, err := r.run(t, "list")
	require.NoError(t, err)
	assert.Contains(t, out, "jev-handoff")
	assert.NotContains(t, out, "corpus")
}

func TestDryRunPrintsThePlanAndMakesNoCall(t *testing.T) {
	t.Parallel()
	r := newRig(t, 5)
	r.generate(t, "jev-handoff")
	out, _, err := r.run(t, "calibrate", "jev-handoff", "--dry-run")
	require.NoError(t, err)
	assert.Zero(t, r.calls)
	for _, want := range []string{"corpus: 5 packages", "calls:", "fields allowed by the data policy: approved_scope, implementation_plan", "projected bytes", "estimated provider usage", "need at least 19", "need an effective 59", "no package is golden yet", "consent term ", "https://provider.example/v1", "projected fields of real packages"} {
		assert.Contains(t, out, want)
	}
}

func TestACalibrationRunNeedsConsent(t *testing.T) {
	t.Parallel()
	r := newRig(t, 5)
	r.generate(t, "jev-handoff")
	_, _, err := r.run(t, "calibrate", "jev-handoff")
	require.ErrorContains(t, err, "--consent")
	assert.Zero(t, r.calls, "without consent no real content is sent")
}

func TestCalibrationRunSharesOneCalibratedState(t *testing.T) {
	t.Parallel()
	r := newRig(t, calibrationCorpusSize)
	r.generate(t, "jev-handoff")
	r.promoteAll(t, "jev-handoff", calibrationCorpusSize)

	t.Run("TestACalibrationRunKeepsEvidenceAndPublishesARecord", func(t *testing.T) {
		out, errOut, err := r.run(t, "calibrate", "jev-handoff", "--consent")

		require.NoError(t, err, "%s\n%s", out, errOut)
		assert.Contains(t, out, "result")
		assert.Contains(t, out, "PASS")
		assert.Contains(t, out, "calibration: cal-")
		assert.Positive(t, r.calls)
		for _, file := range []string{"observations.jsonl", "evidence.json", "corpus.json"} {
			assert.FileExists(t, filepath.Join(r.base, "dojo", "jev-handoff", file))
		}
		assert.FileExists(t, filepath.Join(r.base, "dojo", ".last-run", "jev-handoff", "result.json"))
		assert.NoFileExists(t, filepath.Join(r.base, "dojo", ".last-run", "jev-handoff", "lesson.md"), "a passing run has evidence, not a lesson")
		ids, err := internaldojo.ListCalibrations(r.base)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("TestRecomputeMakesNoCallAndMatchesTheRun", func(t *testing.T) {
		calls := r.calls

		out, _, err := r.run(t, "calibrate", "jev-handoff", "--recompute")
		require.NoError(t, err)
		assert.Equal(t, calls, r.calls)
		assert.Contains(t, out, "PASS")

		checked, _, err := r.run(t, "check", "jev-handoff")
		require.NoError(t, err)
		assert.Contains(t, checked, "calibration gate separation")
	})

	t.Run("TestAnExploratoryRecomputePersistsNothingAndNeedsRecompute", func(t *testing.T) {
		history, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".history.jsonl"))
		require.NoError(t, readErr)

		out, _, err := r.run(t, "calibrate", "jev-handoff", "--recompute", "--exclude-operator", "strip_task_types")
		require.NoError(t, err)
		assert.Contains(t, out, "calibration exploratory what-if")
		assert.Contains(t, out, "nothing was persisted")
		after, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".history.jsonl"))
		require.NoError(t, readErr)
		assert.Equal(t, string(history), string(after), "a what-if never adds to the history")

		_, _, err = r.run(t, "calibrate", "jev-handoff", "--exclude-operator", "strip_task_types", "--dry-run")
		require.ErrorContains(t, err, "needs --recompute")
		_, _, err = r.run(t, "calibrate", "jev-handoff", "--recompute", "--exclude-operator", "nope")
		require.ErrorContains(t, err, "unknown operator")

		judgeOnly, _, err := r.run(t, "calibrate", "jev-handoff", "--recompute", "--judge-only")
		require.NoError(t, err)
		assert.Contains(t, judgeOnly, "calibration exploratory what-if")
		_, _, err = r.run(t, "calibrate", "jev-handoff", "--judge-only", "--dry-run")
		require.ErrorContains(t, err, "needs --recompute")

		_, _, err = r.run(t, "calibrate", "jev-handoff", "--exclude-criterion", "scope_bounded", "--dry-run")
		require.ErrorContains(t, err, "needs --recompute")
		_, _, err = r.run(t, "calibrate", "jev-handoff", "--recompute", "--exclude-criterion", "scope_bounded")
		require.Error(t, err, "an aggregate profile has no per-criterion answers to leave out")
	})
}

func TestCalibrationCorpusJustBelowMinimumFailsForPolicySample(t *testing.T) {
	t.Parallel()
	r := newRig(t, belowCalibrationCorpusSize)
	r.generate(t, "jev-handoff")
	r.promoteAll(t, "jev-handoff", belowCalibrationCorpusSize)
	_, _, err := r.run(t, "calibrate", "jev-handoff", "--consent")
	require.NoError(t, err)

	out, _, err := r.run(t, "calibrate", "jev-handoff", "--recompute", "--exclude-operator", "strip_task_types")
	require.Error(t, err)
	assert.Contains(t, out, "status=insufficient_sample")
	assert.Contains(t, out, "need an effective 59")
}

func TestAFailingJudgeWritesALessonWithTheWeakSpots(t *testing.T) {
	t.Parallel()
	r := newRig(t, calibrationCorpusSize)
	r.generate(t, "jev-handoff")
	r.promoteAll(t, "jev-handoff", calibrationCorpusSize)
	r.judge = func(string, calibration.Fields) calibration.Verdict { return calibration.ScoredVerdict(0.93) }

	out, _, err := r.run(t, "calibrate", "jev-handoff", "--consent")

	require.Error(t, err)
	assert.Contains(t, out, "false_approval@0.80")
	lesson, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".last-run", "jev-handoff", "lesson.md"))
	require.NoError(t, readErr)
	assert.Contains(t, string(lesson), "Weak Spots")
	assert.Contains(t, string(lesson), "false_approval")
}

func TestABudgetStopsTheRun(t *testing.T) {
	t.Parallel()
	r := newRig(t, 10)
	r.generate(t, "jev-handoff")
	_, _, _ = r.run(t, "calibrate", "jev-handoff", "--consent", "--budget", "12", "--repetitions", "1")
	assert.Equal(t, 12, r.calls)
}

func TestCalibrateRefusesAPipelineScenario(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	dir := filepath.Join(r.base, "dojo", "pipeline")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "criteria.yaml"), []byte("scenario: pipeline\nrun_dir: dojo/run\n"), 0o644))
	_, _, err := r.run(t, "calibrate", "pipeline", "--dry-run")
	require.ErrorContains(t, err, "pipeline scenario")
}

func TestPromoteNeedsAReviewerAndABelongingPackage(t *testing.T) {
	t.Parallel()
	r := newRig(t, 2)
	r.generate(t, "jev-handoff")
	_, _, err := r.run(t, "promote", "jev-handoff", "--package", "pkg-000")
	require.Error(t, err, "reviewer and oracle are required flags")
	_, _, err = r.run(t, "promote", "jev-handoff", "--package", "ghost", "--reviewed-by", "me", "--oracle", "good")
	require.ErrorContains(t, err, "not in the corpus")
	out, _, err := r.run(t, "promote", "jev-handoff", "--package", "pkg-000", "--reviewed-by", "me", "--oracle", "reads clean")
	require.NoError(t, err)
	assert.Contains(t, out, "promoted pkg-000")
	_, _, err = r.run(t, "promote", "jev-handoff", "--package", "pkg-000", "--reviewed-by", "me", "--oracle", "again")
	require.Error(t, err)
}

func TestCandidateRecordsOnlyDisagreements(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.signals = []internaldojo.MissionSignal{{Transition: "archivist_to_gate", Answered: true, Approved: true, Confidence: 0.9, Threshold: 0.8, GateOutcome: "accepted"}}
	out, _, err := r.run(t, "candidate", "--mission", "m1")
	require.NoError(t, err)
	assert.Contains(t, out, "no disagreement")

	r.signals = []internaldojo.MissionSignal{{Transition: "archivist_to_gate", Answered: true, Approved: false, Confidence: 0.26, Threshold: 0.8, GateOutcome: "accepted"}}
	out, _, err = r.run(t, "candidate", "--mission", "m2")
	require.NoError(t, err)
	assert.Contains(t, out, internaldojo.CandidateRejectedAccepted)
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "corpus", "candidates.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"mission":"m2"`)
}

func TestAuditSamplesWithoutContentAndRecordsVerdicts(t *testing.T) {
	t.Parallel()
	r := newRig(t, 3)
	r.generate(t, "jev-handoff")
	out, _, err := r.run(t, "audit", "jev-handoff", "--sample", "4")
	require.NoError(t, err)
	assert.Len(t, strings.Split(strings.TrimSpace(out), "\n"), 4)
	assert.NotContains(t, out, "README")
	assert.Contains(t, out, "changed=[")

	_, _, err = r.run(t, "audit", "jev-handoff", "--record", "abc", "--verdict", "agree")
	require.Error(t, err, "a reviewer is required")
	out, _, err = r.run(t, "audit", "jev-handoff", "--record", "abc", "--verdict", "disagree", "--by", "me", "--note", "not a defect")
	require.NoError(t, err)
	assert.Contains(t, out, "recorded disagree")
	assert.FileExists(t, filepath.Join(r.base, "dojo", "jev-handoff", "audit.jsonl"))
}

func TestProposeRecordsAPendingProposal(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	out, _, err := r.run(t, "propose", "swap_sections", "--description", "swap two sections")
	require.NoError(t, err)
	assert.Contains(t, out, "pending")
	_, _, err = r.run(t, "propose", "drop_approved_scope", "--description", "dup")
	require.Error(t, err)
}

func TestUnconfiguredCalibrationDependenciesFailClosed(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.generate(t, "jev-handoff")
	deps := r.deps()
	deps.Calibration = CalibrationDependencies{}
	cmd := New(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"calibrate", "jev-handoff", "--dry-run"})
	require.ErrorContains(t, cmd.Execute(), "not configured")
}

func TestTheIntegrationLayerRunsThePolicyWithNoProviderAndNoConsent(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.policy = []domain.DojoCheckItem{{Label: "calibration policy: approves above the threshold", Passed: true}, {Label: "calibration policy: falls back on timeout", Passed: true}}
	_, _, err := r.run(t, "generate", "policy-case", "--boundary", "package")
	require.NoError(t, err)
	path := filepath.Join(r.base, "dojo", "policy-case", "criteria.yaml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "layer: judge", "layer: integration", 1)), 0o644))

	out, _, err := r.run(t, "calibrate", "policy-case")
	require.NoError(t, err)
	assert.Contains(t, out, "PASS (2 checks)")
	assert.Zero(t, r.calls)
	assert.FileExists(t, filepath.Join(r.base, "dojo", "policy-case", "evidence.json"))
	assert.NoFileExists(t, filepath.Join(r.base, "dojo", ".last-run", "policy-case", "lesson.md"))

	r.policy[1].Passed = false
	_, _, err = r.run(t, "check", "policy-case")
	require.Error(t, err)
	lesson, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".last-run", "policy-case", "lesson.md"))
	require.NoError(t, readErr)
	assert.Contains(t, string(lesson), "policy_violation")
}

func TestCalibrateErrorsAreExplicit(t *testing.T) {
	t.Parallel()
	r := newRig(t, 3)
	_, _, err := r.run(t, "calibrate", "ghost", "--dry-run")
	require.Error(t, err, "an unknown case")

	r.generate(t, "jev-handoff")
	deps := r.deps()
	deps.Calibration.Judge = func(string, string) (calibration.Judge, error) {
		return nil, fmt.Errorf("no provider integration is enabled")
	}
	cmd := New(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"calibrate", "jev-handoff", "--consent"})
	cmd.SetContext(context.Background())
	require.ErrorContains(t, cmd.Execute(), "no provider integration")
}

func TestTheDryRunShowsTheCaseBudgetAndAllSessionsRecomputeUsesEveryRun(t *testing.T) {
	t.Parallel()
	r := newRig(t, 6)
	r.generate(t, "jev-handoff")
	path := filepath.Join(r.base, "dojo", "jev-handoff", "criteria.yaml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "stage: 1", "stage: 1\n    budget: 40", 1)), 0o644))
	out, _, err := r.run(t, "calibrate", "jev-handoff", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, out, "budget 40")

	_, _, _ = r.run(t, "calibrate", "jev-handoff", "--consent", "--repetitions", "1")
	assert.Equal(t, 40, r.calls, "the case budget bounds the run when no flag overrides it")
	_, _, _ = r.run(t, "calibrate", "jev-handoff", "--consent", "--repetitions", "1", "--budget", "10")
	out, _, _ = r.run(t, "calibrate", "jev-handoff", "--recompute", "--all-sessions")
	assert.Contains(t, out, "calibration status", "recomputing every session still evaluates the stored observations")
	assert.NotContains(t, out, "no observations")
}

func TestPromoteAndAuditRefuseAnUnknownCase(t *testing.T) {
	t.Parallel()
	r := newRig(t, 1)
	_, _, err := r.run(t, "promote", "ghost", "--package", "p", "--reviewed-by", "me", "--oracle", "x")
	require.Error(t, err)
	_, _, err = r.run(t, "audit", "ghost")
	require.Error(t, err)
}

func TestCandidateRecordsEveryDisagreementOfAMission(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.signals = []internaldojo.MissionSignal{
		{Transition: "ranger_to_archivist", Answered: true, Approved: false, Confidence: 0.4, Threshold: 0.8, GateOutcome: "accepted"},
		{Transition: "archivist_to_gate", Answered: true, Approved: false, Confidence: 0.3, Threshold: 0.8, GateOutcome: "accepted"},
	}
	out, _, err := r.run(t, "candidate", "--mission", "m3")
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(out, "candidate m3"))
}

func TestGenerateWritesAnIntegrationLayerCaseAndRefusesAnUnknownLayer(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "jev-policy", "--layer", "integration")
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "jev-policy", "criteria.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "layer: integration")
	assert.Contains(t, string(raw), "no provider is called")
	_, _, err = r.run(t, "generate", "other", "--layer", "weird")
	require.Error(t, err)
}

func TestTheDryRunPrintsTheConsentTermAndRunsBoundToItAreRefusedWhenThePlanChanged(t *testing.T) {
	t.Parallel()
	r := newRig(t, 6)
	r.generate(t, "jev-handoff")
	out, _, err := r.run(t, "calibrate", "jev-handoff", "--dry-run")
	require.NoError(t, err)
	term := regexp.MustCompile(`consent term ([0-9a-f]{12})`).FindStringSubmatch(out)
	require.Len(t, term, 2)
	again, _, err := r.run(t, "calibrate", "jev-handoff", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, again, "consent term "+term[1], "the same plan gives the same term")

	_, _, err = r.run(t, "calibrate", "jev-handoff", "--consent", "--consent-term", "000000000000")
	require.ErrorContains(t, err, "the plan changed")
	assert.Zero(t, r.calls, "a refused term sends nothing")

	_, _, _ = r.run(t, "calibrate", "jev-handoff", "--consent", "--consent-term", term[1], "--repetitions", "3")
	assert.Positive(t, r.calls)
	raw, readErr := os.ReadFile(filepath.Join(r.base, "dojo", "jev-handoff", "evidence.json"))
	require.NoError(t, readErr)
	assert.Contains(t, string(raw), `"consent"`)
	assert.Contains(t, string(raw), `"bound": true`)
	assert.Contains(t, string(raw), term[1])
}

func TestASyntheticCaseCalibratesOverFixturesGoldenByConstruction(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	out, _, err := r.run(t, "generate", "jev-pilot", "--synthetic", "--packages", "4")
	require.NoError(t, err)
	assert.Contains(t, out, "criteria.yaml")
	assert.DirExists(t, filepath.Join(r.base, "dojo", "jev-pilot", "fixtures", "done", "synthetic-audit-log"))

	plan, _, err := r.run(t, "calibrate", "jev-pilot", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, plan, "corpus: 4 packages (golden 4")
	assert.Contains(t, plan, "synthetic fixtures (no real package content)")
	assert.Zero(t, r.calls)

	_, _, _ = r.run(t, "calibrate", "jev-pilot", "--consent", "--repetitions", "1")
	assert.Positive(t, r.calls)
	list, _, err := r.run(t, "list")
	require.NoError(t, err)
	assert.Contains(t, list, "jev-pilot")
}

func TestTheLiveLayerNeedsConsentAndACheckNeverCallsOut(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	r.live = []domain.DojoCheckItem{{Label: "live probe authenticates and answers in the documented shape", Passed: true}}
	_, _, err := r.run(t, "generate", "jev-live", "--layer", "live")
	require.NoError(t, err)

	out, _, err := r.run(t, "calibrate", "jev-live", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, out, "synthetic probe and handoff text")
	assert.Contains(t, out, "calls: 3")
	assert.Zero(t, r.liveCalls)

	_, _, err = r.run(t, "calibrate", "jev-live")
	require.ErrorContains(t, err, "--consent")
	assert.Zero(t, r.liveCalls)
	_, _, err = r.run(t, "check", "jev-live")
	require.ErrorContains(t, err, "never calls out")
	assert.Zero(t, r.liveCalls)

	out, _, err = r.run(t, "calibrate", "jev-live", "--consent")
	require.NoError(t, err)
	assert.Equal(t, 1, r.liveCalls)
	assert.Contains(t, out, "PASS")
	raw, readErr := os.ReadFile(filepath.Join(r.base, "dojo", "jev-live", "evidence.json"))
	require.NoError(t, readErr)
	assert.Contains(t, string(raw), `"consent"`)

	r.live = []domain.DojoCheckItem{{Label: "live probe authenticates and answers in the documented shape", Passed: false, Detail: "state authentication_failed"}}
	_, _, err = r.run(t, "calibrate", "jev-live", "--consent")
	require.Error(t, err)
	lesson, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".last-run", "jev-live", "lesson.md"))
	require.NoError(t, readErr)
	assert.Contains(t, string(lesson), "live_check_failed")
}

func TestAV2CaseAsksTheV2ProfileAndSaysSoInItsTerm(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "jev-pilot-v2", "--synthetic", "--packages", "2", "--profile", "handoff-v2")
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "jev-pilot-v2", "criteria.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "profile: handoff-v2")

	out, _, err := r.run(t, "calibrate", "jev-pilot-v2", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, out, "question profile: handoff-v2")

	_, _, err = r.run(t, "generate", "bad", "--profile", "handoff-v9")
	require.Error(t, err)
}

func TestAnAtomicGates3CaseSaysSoInItsSkeletonAndItsPlan(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "atomic", "--synthetic", "--packages", "3", "--profile", "handoff-atomic-v1", "--gate-rule", "gates-3")
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "atomic", "criteria.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "profile: handoff-atomic-v1")
	assert.Contains(t, string(raw), "gate_rule: gates-3")

	out, _, err := r.run(t, "calibrate", "atomic", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, out, "question profile: handoff-atomic-v1")
	assert.Contains(t, out, "gates-3 reads all of them")

	_, _, err = r.run(t, "generate", "bad-rule", "--gate-rule", "gates-9")
	require.Error(t, err)
}

func TestGenerateCanRestrictOperatorsAndRepetitionsAndTheCaseCarriesThem(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "lab", "--synthetic", "--packages", "3", "--repetitions", "2", "--exclude-operator", "strip_task_types")
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(r.base, "dojo", "lab", "criteria.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "repetitions: 2")
	assert.Contains(t, string(raw), "strip_all_task_types")
	assert.NotContains(t, string(raw), "- strip_task_types")
	assert.NotContains(t, string(raw), "stage2")
	assert.DirExists(t, filepath.Join(r.base, "dojo", "lab", "fixtures", "done", "synthetic-audit-log"))
}

func TestCompareReadsTwoCasesOfflineAndFailsClosedWithoutObservations(t *testing.T) {
	t.Parallel()
	r := newRig(t, 10)
	r.generate(t, "a")
	r.generate(t, "b")
	_, _, err := r.run(t, "compare", "a", "b")
	require.ErrorContains(t, err, "no observations")
	_, _, _ = r.run(t, "calibrate", "a", "--consent", "--repetitions", "1") // not golden: a failed case still keeps its observations
	_, _, _ = r.run(t, "calibrate", "b", "--consent", "--repetitions", "1")
	calls := r.calls

	out, _, err := r.run(t, "compare", "a", "b")

	require.NoError(t, err)
	assert.Equal(t, calls, r.calls, "a comparison makes no call")
	assert.Contains(t, out, "laboratory comparison")
	_, _, err = r.run(t, "compare", "a", "ghost")
	require.Error(t, err)
}

func TestAPairCaseIsGeneratedPlannedAssentedRunAndCheckedOffline(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "pairs-lab", "--layer", "pair", "--packages", "12")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(r.base, "dojo", "pairs-lab", "pairs.yaml"))

	plan, _, err := r.run(t, "calibrate", "pairs-lab", "--dry-run")
	require.NoError(t, err)
	for _, want := range []string{"layer: pair", "pairs: 72", "from 12 parents", "calls: 216", "all 3 dimensions", "consent term ", "requirement and task pairs"} {
		assert.Contains(t, plan, want)
	}
	assert.Zero(t, r.pairCalls)

	_, _, err = r.run(t, "calibrate", "pairs-lab")
	require.ErrorContains(t, err, "--consent")
	assert.Zero(t, r.pairCalls)

	out, _, err := r.run(t, "calibrate", "pairs-lab", "--consent")
	require.NoError(t, err)
	assert.Equal(t, 216, r.pairCalls)
	assert.Contains(t, out, "calibration dimension obligation_preserved")
	assert.Contains(t, out, "measured provider usage: 21600 input")
	assert.Contains(t, out, "calibration family weakened_obligation")
	evidence, readErr := os.ReadFile(filepath.Join(r.base, "dojo", "pairs-lab", "evidence.json"))
	require.NoError(t, readErr)
	assert.Contains(t, string(evidence), `"consent"`)
	raw, readErr := os.ReadFile(filepath.Join(r.base, "dojo", "pairs-lab", "pair-observations.jsonl"))
	require.NoError(t, readErr)
	assert.NotContains(t, string(raw), "consumers")

	calls := r.pairCalls
	checked, _, err := r.run(t, "check", "pairs-lab")
	require.NoError(t, err)
	assert.Contains(t, checked, "PASS")
	again, _, err := r.run(t, "calibrate", "pairs-lab", "--recompute")
	require.NoError(t, err)
	assert.Contains(t, again, "PASS")
	assert.Equal(t, calls, r.pairCalls, "check and recompute make no call")
}

func TestABlindPairDimensionFailsTheCaseAndWritesALesson(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "pairs-lab", "--layer", "pair", "--packages", "12")
	require.NoError(t, err)
	deps := r.deps()
	deps.Calibration.PairJudge = func(string) (calibration.PairJudge, error) {
		return calibration.FuncPairJudge(func(p calibration.Pair) calibration.PairVerdict {
			v := r.pairJudge(p)
			v.Scores[calibration.DimObligation] = 0.6
			return v
		}), nil
	}
	cmd := New(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"calibrate", "pairs-lab", "--consent"})
	cmd.SetContext(context.Background())
	require.Error(t, cmd.Execute())
	assert.Contains(t, out.String(), "failing gate separation")
	lesson, readErr := os.ReadFile(filepath.Join(r.base, "dojo", ".last-run", "pairs-lab", "lesson.md"))
	require.NoError(t, readErr)
	assert.Contains(t, string(lesson), "dimension_not_fit")
}

func TestAPairCaseWithoutObservationsOrFixturesFailsClosed(t *testing.T) {
	t.Parallel()
	r := newRig(t, 0)
	_, _, err := r.run(t, "generate", "pairs-lab", "--layer", "pair", "--packages", "2")
	require.NoError(t, err)
	_, _, err = r.run(t, "check", "pairs-lab")
	require.Error(t, err)
	require.NoError(t, os.Remove(filepath.Join(r.base, "dojo", "pairs-lab", "pairs.yaml")))
	_, _, err = r.run(t, "calibrate", "pairs-lab", "--dry-run")
	require.Error(t, err)
}

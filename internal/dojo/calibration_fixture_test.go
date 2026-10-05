package dojo_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/require"
)

const planTemplate = `- [ ] 1.1 [analysis_artifact] Record the answers in design-%[1]d.md; verify D1 is updated.
- [ ] 1.2 [implementation_handoff] Refresh the README %[1]d and relocate the package; verify dojo list and the README agree.
- [ ] 2.1 [implementation_handoff] Build the manifest builder %[1]d; verify tests cover inclusion and determinism.
- [ ] 2.2 [implementation_handoff] Build the catalog %[1]d; verify a golden test per operator. The agent MUST NOT relabel a case.
- [ ] 3.1 [implementation_handoff] Build the evaluator %[1]d; verify it against a scripted judge.
- [ ] 3.2 [analysis_artifact] Report the first calibration %[1]d; verify it states measured values only.
`

type workspace struct {
	base string
}

func newWorkspace(t *testing.T) workspace { t.Helper(); return workspace{base: t.TempDir()} }

func (w workspace) writePackage(t *testing.T, root, name string, variant int, status string) string {
	t.Helper()
	dir := filepath.Join(w.base, root, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	analysis := fmt.Sprintf("---\nmission_id: %s\nmission_status: %s\n---\n\n## affected_scope\n- allowed: docs/**, internal/dojo/**\n- forbidden: internal/domain/**\n", name, status)
	files := map[string]string{"analysis.md": analysis, "proposal.md": "## Why\nx\n", "design.md": "## Context\nx\n", "tasks.md": "---\nacceptance_checks:\n  - \"the checks pass\"\n---\n" + fmt.Sprintf(planTemplate, variant)}
	for file, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644))
	}
	return dir
}

func (w workspace) packages(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		w.writePackage(t, "done", fmt.Sprintf("pkg-%03d", i), i, "documentation_applied")
	}
}

func (w workspace) promoteAll(t *testing.T, cal domain.DojoCalibration) {
	t.Helper()
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	var lines []string
	for _, e := range corpus.Manifest.Entries {
		raw, err := json.Marshal(dojo.Promotion{Digest: e.Digest, Package: e.Name, ReviewedBy: "reviewer", Oracle: "fixture", At: "2026-10-04T00:00:00Z"})
		require.NoError(t, err)
		lines = append(lines, string(raw))
	}
	dir := filepath.Join(w.base, "dojo", dojo.DirCorpus)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "golden.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644))
}

func policyCase() domain.DojoCalibration {
	return domain.DojoCalibration{
		Boundary: calibration.BoundaryPackage, Seed: 1, Repetitions: 2,
		RiskPolicy: domain.DojoRiskPolicy{MaxFalseApproval: 0.05, MaxFalseRejection: 0.15},
	}
}

func criteriaFor(name string, cal domain.DojoCalibration) domain.DojoCriteria {
	return domain.DojoCriteria{Scenario: name, Kind: domain.KindJudgeCalibration, Calibration: &cal}
}

// ruleJudge scores a package high only when scope, task types, verification and
// the implementation handoffs are all intact.
func ruleJudge(_ string, f calibration.Fields) calibration.Verdict {
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

// stubReader reads the package tasks as the plan and the analysis scope text.
type stubReader struct{}

func (stubReader) Package(tasks, analysis string) (calibration.Fields, error) {
	return handoffconsumer.Reader{}.Package(tasks, analysis)
}

func (stubReader) Ranger(path string) (calibration.Fields, error) {
	return handoffconsumer.Reader{}.Ranger(path)
}

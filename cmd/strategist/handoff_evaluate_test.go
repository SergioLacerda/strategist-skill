package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const evalInformationalFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
`

const evalRequiredFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: ["which store?"]
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: false
`

// evalMission builds a .strategist root whose mission m1 waits at the handoff
// challenge, with a refined package carrying facts and tasks.
func evalMission(t *testing.T, facts, tasks string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	testutil.MinimalRoot(t, root)

	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m1"})
	require.NoError(t, err)
	pkg := filepath.Join(filepath.Dir(root), ".analysis", "refined", "m1")
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	files := map[string]string{
		"analysis.md": "---\nmission_id: m1\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tasks,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(pkg, name), []byte(content), 0o644))
	}
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone,
	} {
		_, err = engine.Submit(event)
		require.NoError(t, err, event)
	}
	_, err = engine.Submit(domain.MissionEventGateApproved)
	require.NoError(t, err)
	digest, err := handoff.PackageDigest(pkg)
	require.NoError(t, err)
	status, err := engine.RecordApprovalGatePackageDigest(digest)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	require.NoError(t, saveMission(root, status))
	return root
}

func runEvaluate(t *testing.T, opts handoffEvaluateOptions) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := runArchivistHandoff(cmd, opts)
	return out.String(), err
}

const evalChallenges = `challenges:
  - {id: HC-001, type: objective, source_refs: [G-001], critical: true}
  - {id: HC-002, type: boundary, source_refs: [X-001], critical: true}
  - id: HC-003
    type: classification
    source_refs: [D-001, Q-001]
    critical: true
    expected_classification: {D-001: approved_decision, Q-001: unresolved_question}
  - {id: HC-004, type: gate, source_refs: [approval.required], critical: true, expected_gate_allowed: false}
`

const evalPassingAck = `challenge_refs: [HC-001, HC-002, HC-003, HC-004]
understood_refs: [G-001, X-001, D-001, Q-001, approval.required]
classifications: {D-001: approved_decision, Q-001: unresolved_question}
gate_allowed: false
`

const evalFailingAck = `challenge_refs: [HC-001, HC-002, HC-003, HC-004]
understood_refs: [G-001, X-001, D-001, Q-001, approval.required]
classifications: {D-001: approved_decision, Q-001: approved_decision}
gate_allowed: true
`

func writeEvalFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestHandoffEvaluateOptionValidation(t *testing.T) {
	_, err := runEvaluate(t, handoffEvaluateOptions{})
	require.ErrorContains(t, err, "--mission-id")

	_, err = runEvaluate(t, handoffEvaluateOptions{MissionID: "m1", Challenges: "c.yaml"})
	require.ErrorContains(t, err, "--challenges and --ack must be supplied together")

	_, err = runEvaluate(t, handoffEvaluateOptions{MissionID: "m1", Root: filepath.Join(t.TempDir(), "absent")})
	require.Error(t, err)
}

func TestHandoffEvaluateRecordsAnInformationalSkip(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	assert.Contains(t, out, "outcome: ")
	assert.Contains(t, out, "required: false")
	assert.Contains(t, out, "attempt: 1")
}

func TestHandoffEvaluateRequiredChallengePassesAndFails(t *testing.T) {
	root := evalMission(t, evalRequiredFacts, "- [ ] 1.1 [task_type: implementation_handoff] change the code\n")

	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.ErrorContains(t, err, "handoff_challenge_required")

	failing, err := runEvaluate(t, handoffEvaluateOptions{
		Root: root, MissionID: "m1", RiskLevel: "high",
		Challenges: writeEvalFile(t, "c.yaml", evalChallenges), Ack: writeEvalFile(t, "a.yaml", evalFailingAck),
	})
	require.ErrorContains(t, err, "the mission returned to refinement")
	assert.Contains(t, failing, "required: true")
	assert.Contains(t, failing, "status: ")

	_, status, loadErr := loadMission(root, "m1")
	require.NoError(t, loadErr)
	assert.Equal(t, domain.StateRefinement, status.State)
}

func TestHandoffEvaluateRequiredChallengePasses(t *testing.T) {
	root := evalMission(t, evalRequiredFacts, "- [ ] 1.1 [task_type: implementation_handoff] change the code\n")
	out, err := runEvaluate(t, handoffEvaluateOptions{
		Root: root, MissionID: "m1",
		Challenges: writeEvalFile(t, "c.yaml", evalChallenges), Ack: writeEvalFile(t, "a.yaml", evalPassingAck),
	})
	require.NoError(t, err)
	assert.Contains(t, out, "outcome: passed")
	assert.Contains(t, out, "passed: true")
}

func TestHandoffEvaluateInputFileFailures(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	good := writeEvalFile(t, "good.yaml", evalChallenges)
	ack := writeEvalFile(t, "ack.yaml", evalPassingAck)
	broken := writeEvalFile(t, "broken.yaml", "challenges: [unclosed")
	missing := filepath.Join(t.TempDir(), "absent.yaml")

	cases := map[string]handoffEvaluateOptions{
		"missing confidence summary": {Root: root, MissionID: "m1", ConfidenceSummary: missing},
		"broken confidence summary":  {Root: root, MissionID: "m1", ConfidenceSummary: broken},
		"missing challenges":         {Root: root, MissionID: "m1", Challenges: missing, Ack: ack},
		"broken challenges":          {Root: root, MissionID: "m1", Challenges: broken, Ack: ack},
		"missing ack":                {Root: root, MissionID: "m1", Challenges: good, Ack: missing},
		"broken ack":                 {Root: root, MissionID: "m1", Challenges: good, Ack: broken},
	}
	for name, opts := range cases {
		_, err := runEvaluate(t, opts)
		require.ErrorContains(t, err, "handoff evaluate:", name)
	}

	summary := writeEvalFile(t, "summary.yaml", "policy_version: v1\nclaims: []\nsample_size: 0\ncalibration_status: no_sample\nmissing_record: true\n")
	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1", ConfidenceSummary: summary})
	require.NoError(t, err)
}

func TestHandoffEvaluateUnknownMissionFails(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "ghost"})
	require.ErrorContains(t, err, "handoff evaluate:")

	_, err = recordArchivistHandoffLocked(root, filepath.Join(filepath.Dir(root), ".analysis"), "ghost", livemissionInput())
	require.Error(t, err)
}

func livemissionInput() livemission.ArchivistHandoffInput { return livemission.ArchivistHandoffInput{} }

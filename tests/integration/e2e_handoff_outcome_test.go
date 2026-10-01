//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const informationalPackageFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
`

const requiredPackageFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: ["which store?"]
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: false
`

const informationalPackageTasks = "- [ ] 1.1 [analysis_artifact] record the evidence\n"
const implementationPackageTasks = "- [ ] 1.1 [task_type: implementation_handoff] change the code\n"

// startMissionAtHandoff drives a mission through the real CLI to the state
// that precedes execution and writes its refined package.
func startMissionAtHandoff(t *testing.T, workspace, missionID, facts, tasks string) {
	t.Helper()
	start := runStrategistCLI(t, workspace, "mission", "start", "--mission-id", missionID)
	require.Equal(t, 0, start.exitCode, start.output())
	for _, event := range []string{"bootstrap_done", "intake_done", "discovery_done", "refinement_done"} {
		res := runStrategistCLI(t, workspace, "mission", "submit", "--mission-id", missionID, "--event", event)
		require.Equal(t, 0, res.exitCode, event+": "+res.output())
	}
	dir := filepath.Join(workspace, ".analysis", "refined", missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	files := map[string]string{
		"analysis.md": "---\nmission_id: " + missionID + "\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tasks,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	res := runStrategistCLI(t, workspace, "mission", "submit", "--mission-id", missionID, "--event", "gate_approved")
	require.Equal(t, 0, res.exitCode, res.output())
}

func TestE2E_CLI_HandoffOutcomeGuardsExecutionEntry(t *testing.T) {
	t.Parallel()
	workspace, strategistDir := installedTelemetryWorkspace(t)
	startMissionAtHandoff(t, workspace, "e2e-skip", informationalPackageFacts, informationalPackageTasks)
	submit := func(event string) cliResult {
		return runStrategistCLI(t, workspace, "mission", "submit", "--mission-id", "e2e-skip", "--event", event)
	}

	direct := submit("handoff_challenge_satisfied")
	assert.NotEqual(t, 0, direct.exitCode)
	assert.Contains(t, direct.output(), "handoff_outcome_missing", "the event name alone proves nothing")

	obsolete := submit("handoff_challenge_passed")
	assert.NotEqual(t, 0, obsolete.exitCode)
	assert.Contains(t, obsolete.output(), "unsupported_event")

	verify := runStrategistCLI(t, workspace, "handoff", "verify", "--transition", "archivist_to_sniper", "--mission-id", "e2e-skip")
	assert.NotEqual(t, 0, verify.exitCode)
	assert.Contains(t, verify.output(), "handoff evaluate", "verify no longer decides this transition")

	evaluate := runStrategistCLI(t, workspace, "handoff", "evaluate", "--mission-id", "e2e-skip")
	require.Equal(t, 0, evaluate.exitCode, evaluate.output())
	assert.Contains(t, evaluate.output(), "outcome: skipped")
	assert.FileExists(t, filepath.Join(strategistDir, "missions", "handoff", "e2e-skip", "attempt-001.json"))

	entered := submit("handoff_challenge_satisfied")
	require.Equal(t, 0, entered.exitCode, entered.output())
	assert.FileExists(t, filepath.Join(strategistDir, "missions", "handoff", "e2e-skip", "consumed.json"))

	replay := submit("handoff_challenge_satisfied")
	assert.NotEqual(t, 0, replay.exitCode, "an executing mission cannot enter execution again")
}

func TestE2E_CLI_HandoffEvaluateRefusesPolicyOverridesAndMissingChallenge(t *testing.T) {
	t.Parallel()
	workspace, strategistDir := installedTelemetryWorkspace(t)
	startMissionAtHandoff(t, workspace, "e2e-required", requiredPackageFacts, informationalPackageTasks)

	missing := runStrategistCLI(t, workspace, "handoff", "evaluate", "--mission-id", "e2e-required")
	assert.NotEqual(t, 0, missing.exitCode)
	assert.Contains(t, missing.output(), "handoff_challenge_required")
	assert.NoDirExists(t, filepath.Join(strategistDir, "missions", "handoff", "e2e-required"))

	override := runStrategistCLI(t, workspace, "handoff", "evaluate", "--mission-id", "e2e-required", "--policy", "x.yaml")
	assert.NotEqual(t, 0, override.exitCode, "the policy is derived from the package, never supplied")
}

func TestE2E_CLI_FailedHandoffReturnsToRefinementAndNeedsANewGate(t *testing.T) {
	t.Parallel()
	workspace, strategistDir := installedTelemetryWorkspace(t)
	startMissionAtHandoff(t, workspace, "e2e-loop", requiredPackageFacts, informationalPackageTasks)
	challengesPath := filepath.Join(workspace, "challenges.yaml")
	require.NoError(t, os.WriteFile(challengesPath, []byte("challenges:\n  - id: HC-001\n    type: objective\n    source_refs: [G-001]\n    critical: true\n  - id: HC-002\n    type: boundary\n    source_refs: [X-001]\n    critical: true\n  - id: HC-003\n    type: classification\n    source_refs: [D-001, Q-001]\n    critical: true\n    expected_classification:\n      D-001: approved_decision\n      Q-001: unresolved_question\n  - id: HC-004\n    type: gate\n    source_refs: [approval.required]\n    critical: true\n    expected_gate_allowed: false\n"), 0o644))
	failingAck := filepath.Join(workspace, "ack-fail.yaml")
	require.NoError(t, os.WriteFile(failingAck, []byte("challenge_refs: [HC-001, HC-002, HC-003, HC-004]\nunderstood_refs: [G-001, X-001, D-001, Q-001, approval.required]\nclassifications:\n  D-001: approved_decision\n  Q-001: approved_decision\ngate_allowed: true\n"), 0o644))

	failed := runStrategistCLI(t, workspace, "handoff", "evaluate", "--mission-id", "e2e-loop", "--challenges", challengesPath, "--ack", failingAck)
	assert.NotEqual(t, 0, failed.exitCode)
	assert.Contains(t, failed.output(), "needs a new Approval Gate acceptance")

	status := runStrategistCLI(t, workspace, "mission", "status", "--mission-id", "e2e-loop", "--json")
	require.Equal(t, 0, status.exitCode, status.output())
	assert.Contains(t, status.stdout, `"state":"REFINEMENT"`)
	assert.Contains(t, status.stdout, `"handoff_attempt":1`)
	assert.FileExists(t, filepath.Join(strategistDir, "missions", "handoff", "e2e-loop", "attempt-001.json"))

	direct := runStrategistCLI(t, workspace, "mission", "submit", "--mission-id", "e2e-loop", "--event", "handoff_challenge_satisfied")
	assert.NotEqual(t, 0, direct.exitCode, "execution entry needs the gate again")
}

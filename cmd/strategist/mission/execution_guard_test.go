package mission_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startAtApprovalGate drives a mission through the real submit command up to
// the main Approval Gate, before the package is accepted.
func startAtApprovalGate(t *testing.T, root, id string) {
	t.Helper()
	_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", id)
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone,
	} {
		_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(event))
		require.NoError(t, err, event)
	}
}

// advanceToHandoffChallenge drives a mission through a real gate acceptance
// with the default informational package already present, so the accepted
// package digest is available to the handoff boundary.
func advanceToHandoffChallenge(t *testing.T, root, id string) {
	t.Helper()
	startAtApprovalGate(t, root, id)
	writeRefinedPackage(t, root, id)
	_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(domain.MissionEventGateApproved))
	require.NoError(t, err, domain.MissionEventGateApproved)
}

func TestSubmit_GateApprovalAdvancesRefinedAnalysisStatus(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	startAtApprovalGate(t, root, "m-gate-status")
	writeRefinedPackage(t, root, "m-gate-status")
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	analysisPath := filepath.Join(basePath, "refined", "m-gate-status", "analysis.md")
	raw, err := os.ReadFile(analysisPath)
	require.NoError(t, err)
	raw = []byte(strings.Replace(string(raw), "mission_status: gate_analysis_accepted", "mission_status: archivist_done", 1))
	require.NoError(t, os.WriteFile(analysisPath, raw, 0o600))

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-gate-status", "--event", string(domain.MissionEventGateApproved))
	require.NoError(t, err)

	updated, err := os.ReadFile(analysisPath)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "mission_status: gate_analysis_accepted")
}

func advanceToHandoffChallengeWithPackage(t *testing.T, root, id, facts, tasks string) {
	t.Helper()
	startAtApprovalGate(t, root, id)
	writeTypedPackage(t, root, id, facts, tasks)
	_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(domain.MissionEventGateApproved))
	require.NoError(t, err, domain.MissionEventGateApproved)
}

const informationalFacts = `handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
`

const analysisOnlyTasks = "- [ ] 1.1 [analysis_artifact] record the evidence\n"

// writeRefinedPackage writes a valid, informational (skippable) Archivist package.
func writeRefinedPackage(t *testing.T, root, id string) {
	t.Helper()
	writeTypedPackage(t, root, id, informationalFacts, analysisOnlyTasks)
}

// writeTypedPackage writes a package whose analysis.md carries facts (the YAML
// block, "" for none) and whose tasks.md is tasks.
func writeTypedPackage(t *testing.T, root, id, facts, tasks string) {
	t.Helper()
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	dir := filepath.Join(basePath, "refined", id)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	files := map[string]string{
		"analysis.md": "---\nmission_id: " + id + "\nmission_status: gate_analysis_accepted\n" + facts + "---\n\n## mission_objective\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tasks,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
}

func missionStatus(t *testing.T, root, id string) domain.MissionEngineStatus {
	t.Helper()
	out, err := runLifecycle(t, mission.NewStatus, "--root", root, "--mission-id", id)
	require.NoError(t, err)
	return decodeStatus(t, out)
}

// evaluateHandoff records the mission's Archivist-to-Sniper outcome through the
// production composition, saving the mission state exactly as `handoff
// evaluate` does.
func evaluateHandoff(t *testing.T, root, id string, input livemission.ArchivistHandoffInput) (livemission.ArchivistHandoffResult, error) {
	t.Helper()
	deps := lifecycleDeps(t)
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	engine, _, err := deps.Load(root, id)
	require.NoError(t, err)
	evaluation, status, changed, evalErr := livemission.RecordArchivistHandoff(root, basePath, engine, input)
	if changed {
		require.NoError(t, deps.Save(root, status))
	}
	return evaluation, evalErr
}

func TestSubmit_EnteringExecutionRequiresPipelineEvidence(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-guard")
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(filepath.Join(basePath, "refined", "m-guard")))

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-guard", "--event", string(domain.MissionEventHandoffSatisfied))
	require.Error(t, err)
	assert.Contains(t, err.Error(), domain.PipelineBypassDetectedReason)
	assert.Contains(t, err.Error(), "expected_phase=ranger")

	out, err := runLifecycle(t, mission.NewStatus, "--root", root, "--mission-id", "m-guard")
	require.NoError(t, err)
	assert.Equal(t, domain.StateHandoffChallenge, decodeStatus(t, out).State, "a blocked entry leaves the mission where it was")

	writeRefinedPackage(t, root, "m-guard")
	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-guard", "--event", string(domain.MissionEventHandoffSatisfied))
	require.ErrorContains(t, err, "handoff_outcome_missing", "a complete package without a recorded outcome still cannot enter execution")

	_, err = evaluateHandoff(t, root, "m-guard", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	out, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-guard", "--event", string(domain.MissionEventHandoffSatisfied))
	require.NoError(t, err)
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)
}

// TestSubmit_HandoffPassedRecordsSniperClaim covers ADR-0057 § A1
// (design.md Batch A / task 2.3): the real `mission submit --event
// handoff_challenge_satisfied` path records a Sniper claim for each declared
// documentation_target, without the operator having to invoke a separate
// command — the fix for F-H1's previously-unreachable collision tripwire.
func TestSubmit_HandoffPassedRecordsSniperClaim(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallengeWithPackage(t, root, "m-claim", informationalFacts, "- [ ] 1.1 [documentation_target] Write `docs/example-target.md` for this test.\n")
	_, err := evaluateHandoff(t, root, "m-claim", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-claim", "--event", string(domain.MissionEventHandoffSatisfied))
	require.NoError(t, err)

	records, err := telemetry.ReadRecentSniperClaims(telemetry.SniperClaimHistoryPath(root), time.Now(), telemetry.SniperClaimWindow)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "m-claim", records[0].MissionID)
	assert.Equal(t, "docs/example-target.md", records[0].TargetPath)
}

// TestSubmit_HandoffPassedWithNoDocumentationTargetRecordsNoClaim confirms an
// analysis-only accepted package (writeRefinedPackage's empty tasks.md)
// records nothing — there is no Sniper target to claim.
func TestSubmit_HandoffPassedWithNoDocumentationTargetRecordsNoClaim(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-no-claim")
	writeRefinedPackage(t, root, "m-no-claim")
	_, err := evaluateHandoff(t, root, "m-no-claim", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-no-claim", "--event", string(domain.MissionEventHandoffSatisfied))
	require.NoError(t, err)

	records, err := telemetry.ReadRecentSniperClaims(telemetry.SniperClaimHistoryPath(root), time.Now(), telemetry.SniperClaimWindow)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestSubmit_ScoutRouteDecisionNarrowsTheEvidenceRegime(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-short")
	decision := `{"mission_id":"m-short","request_category":"general","selected_route":"implementation_short_route","route_reason":"refined","route_confidence":0.9,"evidence_state":"explicit","fallback_route":"full_pipeline"}`

	cmd := mission.NewRoute(lifecycleDeps(t))
	cmd.SetIn(strings.NewReader(decision))
	cmd.SetArgs([]string{"--root", root, "--mission-id", "m-short"})
	require.NoError(t, cmd.Execute())

	writeRefinedPackage(t, root, "m-short")
	_, err := evaluateHandoff(t, root, "m-short", livemission.ArchivistHandoffInput{})
	require.NoError(t, err)
	out, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-short", "--event", string(domain.MissionEventHandoffSatisfied))
	require.NoError(t, err, "the short route needs the approved gate and a recorded handoff outcome")
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)
}

func TestRoute_RejectsDecisionForAnotherMission(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	decision := `{"mission_id":"someone-else","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.9,"evidence_state":"explicit","fallback_route":"full_pipeline"}`

	cmd := mission.NewRoute(lifecycleDeps(t))
	cmd.SetIn(strings.NewReader(decision))
	cmd.SetArgs([]string{"--root", root, "--mission-id", "m-route"})

	require.Error(t, cmd.Execute())
}

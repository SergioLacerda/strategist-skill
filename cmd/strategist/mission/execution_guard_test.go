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
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// advanceToHandoffChallenge drives a mission through the real submit command
// up to the state that precedes execution.
func advanceToHandoffChallenge(t *testing.T, root, id string) {
	t.Helper()
	_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", id)
	require.NoError(t, err)
	for _, event := range []domain.MissionEngineEvent{
		domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, domain.MissionEventDiscoveryDone,
		domain.MissionEventRefinementDone, domain.MissionEventGateApproved,
	} {
		_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(event))
		require.NoError(t, err, event)
	}
}

func writeRefinedPackage(t *testing.T, root, id string) {
	t.Helper()
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	dir := filepath.Join(basePath, "refined", id)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	for _, name := range []string{"analysis.md", "proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o600))
	}
}

func TestSubmit_EnteringExecutionRequiresPipelineEvidence(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-guard")

	_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-guard", "--event", string(domain.MissionEventHandoffPassed))
	require.Error(t, err)
	assert.Contains(t, err.Error(), domain.PipelineBypassDetectedReason)
	assert.Contains(t, err.Error(), "expected_phase=ranger")

	out, err := runLifecycle(t, mission.NewStatus, "--root", root, "--mission-id", "m-guard")
	require.NoError(t, err)
	assert.Equal(t, domain.StateHandoffChallenge, decodeStatus(t, out).State, "a blocked entry leaves the mission where it was")

	writeRefinedPackage(t, root, "m-guard")
	out, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-guard", "--event", string(domain.MissionEventHandoffPassed))
	require.NoError(t, err)
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)
}

// TestSubmit_HandoffPassedRecordsSniperClaim covers ADR-0057 § A1
// (design.md Batch A / task 2.3): the real `mission submit --event
// handoff_challenge_passed` path records a Sniper claim for each declared
// documentation_target, without the operator having to invoke a separate
// command — the fix for F-H1's previously-unreachable collision tripwire.
func TestSubmit_HandoffPassedRecordsSniperClaim(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-claim")

	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	dir := filepath.Join(basePath, "refined", "m-claim")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	for _, name := range []string{"analysis.md", "proposal.md", "design.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(
		"- [ ] 1.1 [documentation_target] Write `docs/example-target.md` for this test.\n",
	), 0o600))

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-claim", "--event", string(domain.MissionEventHandoffPassed))
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

	_, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-no-claim", "--event", string(domain.MissionEventHandoffPassed))
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

	out, err := runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-short", "--event", string(domain.MissionEventHandoffPassed))
	require.NoError(t, err, "the short route needs only the approved gate")
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

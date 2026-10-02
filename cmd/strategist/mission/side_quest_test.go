package mission_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sideQuestPackage(t *testing.T, root, id string) {
	t.Helper()
	writeTypedPackage(t, root, id, requiredFacts, sprintfTasks(id))
}

func sprintfTasks(id string) string {
	return "- [ ] 1.1 [task_type: implementation_handoff] change the code\n" +
		"```yaml\nside_quests_approved:\n  - {id: OA-ADR-" + id + ", description: record the decision, strategy: execute_together, status: sq_pending}\n```\n"
}

func sideQuestGate(t *testing.T, id string) string {
	t.Helper()
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToApprovalGate(t, lockedLifecycleDeps(t), root, id)
	sideQuestPackage(t, root, id)
	return root
}

func prepareSideQuestCmd(t *testing.T, canonical string, newCmd func(mission.LifecycleDependencies) *cobra.Command, args ...string) (*cobra.Command, *safeBuffer) {
	t.Helper()
	deps := lockedLifecycleDeps(t)
	deps.ADRCanonicalPath = func(string) (string, error) { return canonical, nil }
	cmd := newCmd(deps)
	out := &safeBuffer{}
	cmd.SetOut(out)
	cmd.SetErr(&safeBuffer{})
	cmd.SetArgs(args)
	return cmd, out
}

func runSideQuestCmd(t *testing.T, canonical string, newCmd func(mission.LifecycleDependencies) *cobra.Command, args ...string) (string, error) {
	t.Helper()
	cmd, out := prepareSideQuestCmd(t, canonical, newCmd, args...)
	err := cmd.Execute()
	return out.String(), err
}

type safeBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func acceptOAADR(t *testing.T, root, id, canonical string) error {
	t.Helper()
	_, err := runSideQuestCmd(t, canonical, mission.NewAcceptSideQuest, "--root", root, "--mission-id", id, "--side-quest", "OA-ADR-"+id)
	return err
}

func submitEvent(t *testing.T, root, id string, event domain.MissionEngineEvent) error {
	t.Helper()
	_, err := runSideQuestCmd(t, "", mission.NewSubmit, "--root", root, "--mission-id", id, "--event", string(event))
	return err
}

func TestAcceptedOAADRMakesAnalysisOnlyEventsRejectedAndLeavesStateUnchanged(t *testing.T) {
	root := sideQuestGate(t, "m-oa")
	require.NoError(t, acceptOAADR(t, root, "m-oa", ""))

	for _, event := range []domain.MissionEngineEvent{domain.MissionEventGateApprovedAnalysisOnly, domain.MissionEventHandoffNotApplicable} {
		err := submitEvent(t, root, "m-oa", event)
		require.ErrorContains(t, err, "gate_event_conflicts_with_accepted_side_quest", event)
		assert.Equal(t, domain.StateApprovalGate, missionStatus(t, root, "m-oa").State, "rejection leaves state unchanged")
	}
}

func TestAcceptedOAADREntersTheHandoffThroughGateApprovedAndSealsTheRecord(t *testing.T) {
	root := sideQuestGate(t, "m-oa")
	require.NoError(t, acceptOAADR(t, root, "m-oa", ""))
	require.NoError(t, submitEvent(t, root, "m-oa", domain.MissionEventGateApproved))
	assert.Equal(t, domain.StateHandoffChallenge, missionStatus(t, root, "m-oa").State)

	rec, found, err := livemission.LoadAcceptedSideQuest(root, "m-oa")
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, rec.Sealed)

	_, err = submitSatisfied(t, root, "m-oa")
	require.ErrorContains(t, err, "handoff_outcome_missing", "the main handoffs still need handoff evaluate")

	_, err = evaluateHandoff(t, root, "m-oa", passingChallenge())
	require.NoError(t, err)
	out, err := submitSatisfied(t, root, "m-oa")
	require.NoError(t, err)
	assert.Equal(t, domain.StateExecution, decodeStatus(t, out).State)

	require.ErrorContains(t, acceptOAADR(t, root, "m-oa", ""), "side_quest_wrong_state", "the decision window closed with the gate")
	_, err = runSideQuestCmd(t, "", mission.NewDeclineSideQuest, "--root", root, "--mission-id", "m-oa", "--side-quest", "OA-ADR-m-oa")
	require.Error(t, err, "a decision cannot change after the gate event")
}

func TestOrdinaryAnalysisOnlyPackageIsUnchanged(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToApprovalGate(t, lockedLifecycleDeps(t), root, "m-plain")
	writeRefinedPackage(t, root, "m-plain")
	require.NoError(t, submitEvent(t, root, "m-plain", domain.MissionEventGateApprovedAnalysisOnly))
	assert.Equal(t, domain.StateDoneAnalysis, missionStatus(t, root, "m-plain").State)
	_, found, err := livemission.LoadAcceptedSideQuest(root, "m-plain")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestDeclinedOAADRKeepsAnalysisOnlyAvailableAndRefusesATarget(t *testing.T) {
	root := sideQuestGate(t, "m-dec")
	_, err := runSideQuestCmd(t, "", mission.NewDeclineSideQuest, "--root", root, "--mission-id", "m-dec", "--side-quest", "OA-ADR-m-dec")
	require.NoError(t, err)
	require.NoError(t, submitEvent(t, root, "m-dec", domain.MissionEventGateApprovedAnalysisOnly))

	_, err = runSideQuestCmd(t, "", mission.NewADRTarget, "--root", root, "--mission-id", "m-dec")
	require.Error(t, err)
}

func TestSideQuestCommandsRejectInvalidRequestsWithoutChangingState(t *testing.T) {
	root := sideQuestGate(t, "m-bad")
	for name, args := range map[string][]string{
		"foreign id": {"--side-quest", "OA-ADR-other"},
		"unknown id": {"--side-quest", "SQ-001"},
		"missing id": {},
	} {
		_, err := runSideQuestCmd(t, "", mission.NewAcceptSideQuest, append([]string{"--root", root, "--mission-id", "m-bad"}, args...)...)
		require.Error(t, err, name)
	}
	require.Error(t, acceptOAADR(t, root, "m-bad", "../outside"), "invalid destination")
	require.Error(t, acceptOAADR(t, root, "m-bad", "/abs"), "absolute destination")
	_, found, err := livemission.LoadAcceptedSideQuest(root, "m-bad")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, domain.StateApprovalGate, missionStatus(t, root, "m-bad").State)

	writeTypedPackage(t, root, "m-bad", requiredFacts, "- [ ] 1.1 [task_type: implementation_handoff] change the code\n")
	require.Error(t, acceptOAADR(t, root, "m-bad", ""), "a package that does not list the OA-ADR")
}

func TestAcceptSideQuestRequiresTheApprovalGate(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	advanceToHandoffChallenge(t, root, "m-late")
	sideQuestPackage(t, root, "m-late")
	require.ErrorContains(t, acceptOAADR(t, root, "m-late", ""), "side_quest_wrong_state")
}

// sideQuestInExecution drives an accepted OA-ADR mission to EXECUTION through
// the real gate, handoff evaluation and entry commands.
func sideQuestInExecution(t *testing.T, id, canonical string) string {
	t.Helper()
	root := sideQuestGate(t, id)
	require.NoError(t, acceptOAADR(t, root, id, canonical))
	_, err := runSideQuestCmd(t, canonical, mission.NewADRTarget, "--root", root, "--mission-id", id)
	require.ErrorContains(t, err, "adr_target_wrong_state")

	require.NoError(t, submitEvent(t, root, id, domain.MissionEventGateApproved))
	_, err = evaluateHandoff(t, root, id, passingChallenge())
	require.NoError(t, err)
	_, err = submitSatisfied(t, root, id)
	require.NoError(t, err)
	return root
}

// concurrentADRTargets runs n adr-target commands at once and returns the path
// each reported ("" when a call failed).
func concurrentADRTargets(t *testing.T, root, id, canonical string, n int) []string {
	t.Helper()
	paths := make([]string, n)
	var wg sync.WaitGroup
	for i := range paths {
		cmd, out := prepareSideQuestCmd(t, canonical, mission.NewADRTarget, "--root", root, "--mission-id", id, "--slug", "Record the Decision")
		wg.Add(1)
		go func() {
			defer wg.Done()
			paths[i] = executeADRTarget(cmd, out)
		}()
	}
	wg.Wait()
	return paths
}

func executeADRTarget(cmd *cobra.Command, out *safeBuffer) string {
	if cmd.Execute() != nil {
		return ""
	}
	var res mission.ADRTargetResult
	if json.Unmarshal([]byte(out.String()), &res) != nil {
		return ""
	}
	return res.Path
}

func TestADRTargetCanonicalNumberingIsReservedOnceAndClaimed(t *testing.T) {
	const want = "docs/adr/0042-record-the-decision.md"
	root := sideQuestInExecution(t, "m-adr", "docs/adr")
	adrDir := filepath.Join(filepath.Dir(root), "docs", "adr")
	require.NoError(t, os.MkdirAll(adrDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(adrDir, "0041-previous.md"), nil, 0o600))

	for _, p := range concurrentADRTargets(t, root, "m-adr", "docs/adr", 4) {
		assert.Equal(t, want, p, "concurrent calls agree on one reserved path")
	}
	claims, err := telemetry.ReadRecentSniperClaims(telemetry.SniperClaimHistoryPath(root), time.Now(), telemetry.SniperClaimWindow)
	require.NoError(t, err)
	require.Len(t, claims, 1, "exactly one claim for the reserved path")
	assert.Equal(t, want, claims[0].TargetPath)
}

func TestInterruptedGateEventStillHonorsTheRecord(t *testing.T) {
	root := sideQuestGate(t, "m-crash")
	require.NoError(t, acceptOAADR(t, root, "m-crash", ""))
	// The record exists but the gate event was never submitted: a later event
	// still sees it, and the mission has not moved.
	assert.Equal(t, domain.StateApprovalGate, missionStatus(t, root, "m-crash").State)
	require.ErrorContains(t, submitEvent(t, root, "m-crash", domain.MissionEventGateApprovedAnalysisOnly), "gate_event_conflicts")
	require.NoError(t, submitEvent(t, root, "m-crash", domain.MissionEventGateApproved))
	rec, _, err := livemission.LoadAcceptedSideQuest(root, "m-crash")
	require.NoError(t, err)
	assert.True(t, rec.Sealed)
}

func TestSideQuestPackageResolvesThroughActiveBasePath(t *testing.T) {
	root := sideQuestGate(t, "m-base")
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(basePath, "refined", "m-base", "tasks.md"))
}

func TestConcurrentAcceptancesConvergeOnOneRecord(t *testing.T) {
	root := sideQuestGate(t, "m-race")
	var wg sync.WaitGroup
	errs := make([]error, 6)
	for i := range errs {
		cmd, _ := prepareSideQuestCmd(t, "", mission.NewAcceptSideQuest, "--root", root, "--mission-id", "m-race", "--side-quest", "OA-ADR-m-race")
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = cmd.Execute()
		}()
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}
	rec, found, err := livemission.LoadAcceptedSideQuest(root, "m-race")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, livemission.SideQuestAccepted, rec.Decision)
}

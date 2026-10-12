package mission

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deferredFixture is a mission whose package lists the three opportunities.
func deferredFixture(t *testing.T, missionID string) sqFixture {
	t.Helper()
	f := newSQFixture(t, missionID)
	tasks := "# t\n```yaml\nside_quests_approved:\n" +
		"  - {id: OA-ADR-" + missionID + ", description: d, strategy: execute_later, status: proposed}\n" +
		"  - {id: OA-RUNBOOK-" + missionID + ", description: d, strategy: execute_later, status: proposed}\n" +
		"  - {id: OA-CHEST-" + missionID + ", description: d, strategy: execute_later, status: proposed}\n```\n"
	require.NoError(t, os.WriteFile(filepath.Join(f.base, "refined", missionID, "tasks.md"), []byte(tasks), 0o600))
	return f
}

func (f sqFixture) accept(t *testing.T, missionID, sideQuestID, canonical string) {
	t.Helper()
	_, err := DecideSideQuest(SideQuestDecision{
		StrategistRoot: f.root, BasePath: f.base, MissionID: missionID, SideQuestID: sideQuestID,
		Decision: SideQuestAccepted, State: domain.StateApprovalGate, CanonicalPath: canonical, Now: time.Now(),
	})
	require.NoError(t, err)
}

func TestAcceptedOAChestAloneAllowsAnalysisOnlyGateEvents(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAChestSideQuestID("m1"), "")
	for _, event := range []domain.MissionEngineEvent{domain.MissionEventGateApprovedAnalysisOnly, domain.MissionEventHandoffNotApplicable} {
		assert.NoError(t, RequireNoAcceptedSideQuest(f.root, "m1", event), string(event))
	}
}

func TestAcceptedOAChestIsDeferredWithAnExplicitNotice(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAChestSideQuestID("m1"), "")
	snapshot, err := ResolveExecutionSnapshot(f.root, f.base, "m1", []string{"docs/guide.md"}, time.Now())
	require.NoError(t, err)
	require.Len(t, snapshot.Groups, 1, "only the explicit documentation target is a group")
	assert.Equal(t, domain.ExecutionTargetKindDocumentation, snapshot.Groups[0].Kind)
	require.Len(t, snapshot.Deferred, 1)
	assert.Equal(t, OAChestSideQuestID("m1"), snapshot.Deferred[0].SideQuestID)
	assert.Equal(t, SideQuestKindOAChest, snapshot.Deferred[0].Kind)
	assert.Equal(t, domain.DeferredReasonNoExecutionResolver, snapshot.Deferred[0].Reason)
}

func TestAcceptedOAChestAloneYieldsNoGroupsButStillReportsTheNotice(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAChestSideQuestID("m1"), "")
	snapshot, err := ResolveExecutionSnapshot(f.root, f.base, "m1", nil, time.Now())
	require.NoError(t, err)
	assert.Empty(t, snapshot.Groups)
	assert.Len(t, snapshot.Deferred, 1)
}

func TestAcceptedOAChestNextToAcceptedADRMaterializesOnlyTheADR(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAADRSideQuestID("m1"), "docs/adr")
	f.accept(t, "m1", OAChestSideQuestID("m1"), "")
	snapshot, err := ResolveExecutionSnapshot(f.root, f.base, "m1", nil, time.Now())
	require.NoError(t, err)
	require.Len(t, snapshot.Groups, 1)
	assert.Equal(t, domain.ExecutionTargetKindADR, snapshot.Groups[0].Kind)
	require.Len(t, snapshot.Deferred, 1)
	assert.Error(t, RequireNoAcceptedSideQuest(f.root, "m1", domain.MissionEventGateApprovedAnalysisOnly),
		"the accepted ADR still forbids the analysis-only exit")
}

func TestAcceptedADRStillRejectsAnalysisOnlyEvents(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAADRSideQuestID("m1"), "docs/adr")
	assert.ErrorIs(t, RequireNoAcceptedSideQuest(f.root, "m1", domain.MissionEventGateApprovedAnalysisOnly), ErrGateEventConflictsWithSideQuest)
}

func TestAcceptedRunbookStillRejectsAnalysisOnlyEvents(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OARunbookSideQuestID("m1"), "docs/runbooks")
	assert.ErrorIs(t, RequireNoAcceptedSideQuest(f.root, "m1", domain.MissionEventHandoffNotApplicable), ErrGateEventConflictsWithSideQuest)
}

func TestOtherResolutionErrorsStillFailClosedAlongsideADeferredChest(t *testing.T) {
	f := deferredFixture(t, "m1")
	f.accept(t, "m1", OAADRSideQuestID("m1"), "docs/adr")
	f.accept(t, "m1", OAChestSideQuestID("m1"), "")
	require.NoError(t, os.MkdirAll(filepath.Join(f.workspace, "docs", "adr"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(f.workspace, "docs", "adr", "m1-adr.md"), []byte("existing"), 0o600))
	_, err := ResolveExecutionSnapshot(f.root, f.base, "m1", nil, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adr_target_exists")
}

func TestRegistryReportsWhichKindsItCanResolve(t *testing.T) {
	registry, err := builtinResolverRegistry("", "", time.Now())
	require.NoError(t, err)
	origin := domain.ExecutionTargetOriginAcceptedSideQuest
	assert.True(t, registry.Has(domain.ExecutionTargetKindADR, origin))
	assert.True(t, registry.Has("runbook", origin))
	assert.False(t, registry.Has(SideQuestKindOAChest, origin))
}

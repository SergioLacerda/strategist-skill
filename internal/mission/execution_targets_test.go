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

func TestResolveExecutionTargetsProjectsExplicitTargetsAndAcceptedADR(t *testing.T) {
	f := newSQFixture(t, "m1")
	_, err := f.decide("m1", SideQuestAccepted, "docs/adr", domain.StateApprovalGate)
	require.NoError(t, err)

	first, err := ResolveExecutionTargets(f.root, f.base, "m1", []string{"docs/guide.md", "docs/guide.md"}, time.Unix(100, 0))
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, domain.ExecutionTarget{
		Path: "docs/guide.md", Kind: domain.ExecutionTargetKindDocumentation,
		Origin: domain.ExecutionTargetOriginRefinedPackage,
	}, first[0])
	assert.Equal(t, domain.ExecutionTarget{
		Path: "docs/adr/m1-adr.md", Kind: domain.ExecutionTargetKindADR,
		Origin: domain.ExecutionTargetOriginAcceptedSideQuest, SideQuestID: OAADRSideQuestID("m1"),
	}, first[1])

	replayed, err := ResolveExecutionTargets(f.root, f.base, "m1", []string{"docs/guide.md"}, time.Unix(200, 0))
	require.NoError(t, err)
	assert.Equal(t, first, replayed)
	rec, found, err := LoadSideQuest(f.root, "m1", OAADRSideQuestID("m1"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "docs/adr/m1-adr.md", rec.ReservedPath)
	assert.False(t, rec.Claimed, "execution resolution reserves the ADR but does not duplicate its materialization claim")
}

func TestResolveExecutionTargetsAbsentOrDeclinedADRRemainsAnalysisOnly(t *testing.T) {
	f := newSQFixture(t, "m1")
	resolved, err := ResolveExecutionTargets(f.root, f.base, "m1", nil, time.Now())
	require.NoError(t, err)
	assert.Empty(t, resolved)

	_, err = f.decide("m1", SideQuestDeclined, "", domain.StateApprovalGate)
	require.NoError(t, err)
	resolved, err = ResolveExecutionTargets(f.root, f.base, "m1", nil, time.Now())
	require.NoError(t, err)
	assert.Empty(t, resolved)
}

func TestResolveExecutionTargetsProjectsAcceptedRunbookGroup(t *testing.T) {
	f := newSQFixture(t, "m1")
	require.NoError(t, os.WriteFile(filepath.Join(f.base, "refined", "m1", "tasks.md"), []byte("# t\n```yaml\nside_quests_approved:\n  - {id: OA-RUNBOOK-m1, description: procedure, strategy: execute_later, status: proposed}\n```\n"), 0o600))
	_, err := DecideSideQuest(SideQuestDecision{
		StrategistRoot: f.root, BasePath: f.base, MissionID: "m1", SideQuestID: OARunbookSideQuestID("m1"),
		Decision: SideQuestAccepted, State: domain.StateApprovalGate, CanonicalPath: "docs/runbooks", Now: time.Now(),
	})
	require.NoError(t, err)

	targets, err := ResolveExecutionTargets(f.root, f.base, "m1", nil, time.Now())
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "docs/runbooks/m1-runbook.md", targets[0].Path)
	assert.Equal(t, "docs/runbooks/m1-runbook.meta.json", targets[1].Path)
}

func TestResolveExecutionTargetsExistingADRFailsClosed(t *testing.T) {
	f := newSQFixture(t, "m1")
	_, err := f.decide("m1", SideQuestAccepted, "docs/adr", domain.StateApprovalGate)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(f.workspace, "docs", "adr"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(f.workspace, "docs", "adr", "m1-adr.md"), []byte("existing"), 0o600))

	_, err = ResolveExecutionTargets(f.root, f.base, "m1", nil, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adr_target_exists")
	rec, found, loadErr := LoadSideQuest(f.root, "m1", OAADRSideQuestID("m1"))
	require.NoError(t, loadErr)
	require.True(t, found)
	assert.Empty(t, rec.ReservedPath)
}

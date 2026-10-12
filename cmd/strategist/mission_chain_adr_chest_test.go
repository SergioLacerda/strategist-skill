package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adrChainResultWithChest() string {
	const adrEntry = "  - {id: OA-ADR-" + archivistMissionID + ", description: record the decision, strategy: execute_together, status: sq_pending}\n"
	chestEntry := "  - {id: OA-CHEST-" + archivistMissionID + ", description: register the chest, strategy: execute_later, status: sq_pending}\n"
	return strings.Replace(adrChainResult, adrEntry, adrEntry+chestEntry, 1)
}

// TestWritingPlansOAADRChainDefersChestAndClosesADR proves the real mixed
// chain: writing-plans emits both accepted intents, the ADR becomes a Sniper
// target, the chest remains an explicit deferred notice, and completion needs
// no chest artifact or ledger proof.
func TestWritingPlansOAADRChainDefersChestAndClosesADR(t *testing.T) {
	root, basePath := chainWorkspace(t)
	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))
	require.NoError(t, err)
	_, err = completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, adrChainResultWithChest()))
	require.NoError(t, err)

	require.NoError(t, chainSubmit(t, root, domain.MissionEventRefinementDone))
	chainAcceptADR(t, root)
	chainAcceptSideQuest(t, root, "OA-CHEST-"+archivistMissionID)
	require.NoError(t, chainSubmit(t, root, domain.MissionEventGateApproved))
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runArchivistHandoff(cmd, handoffEvaluateOptions{Root: root, MissionID: archivistMissionID}))
	require.NoError(t, chainSubmit(t, root, domain.MissionEventHandoffSatisfied))

	sniper, err := buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{
		Root: root, BasePath: basePath, MissionID: archivistMissionID, Role: "sniper", Slot: "execution",
	})
	require.NoError(t, err)
	targets, ok := sniper.Input["execution_targets"].([]domain.ExecutionTarget)
	require.True(t, ok)
	require.Len(t, targets, 1)
	assert.Equal(t, domain.ExecutionTargetKindADR, targets[0].Kind)
	assert.Equal(t, "OA-ADR-"+archivistMissionID, targets[0].SideQuestID)

	snapshot, ok := sniper.Input["execution_snapshot"].(domain.ExecutionSnapshot)
	require.True(t, ok)
	require.Len(t, snapshot.Deferred, 1)
	assert.Equal(t, "OA-CHEST-"+archivistMissionID, snapshot.Deferred[0].SideQuestID)
	assert.Equal(t, livemission.SideQuestKindOAChest, snapshot.Deferred[0].Kind)
	assert.Equal(t, domain.DeferredReasonNoExecutionResolver, snapshot.Deferred[0].Reason)

	materializeChainReport(t, basePath)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), targets[0].Path), []byte("# ADR\n"), 0o600))
	require.NoError(t, telemetry.AppendSniperMaterialization(telemetry.SniperMaterializationHistoryPath(root), telemetry.SniperMaterializationRecord{
		MissionID: archivistMissionID, BasePath: basePath, TargetPath: targets[0].Path, MaterializedAt: time.Now().UTC(),
	}))
	_, err = completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
		Root: root, BasePath: basePath, RequestID: sniper.RequestID, Adapter: domain.ExecutionAdapterCurrentHost,
		Completion: domain.MissionInvocationCompletion{RequestID: sniper.RequestID, Result: sniperChainResult},
	})
	require.NoError(t, err)

	record, found, err := livemission.LoadAcceptedSideQuest(root, archivistMissionID)
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, record.Sealed)
	chestLedger := telemetry.SniperMaterializationHistoryPath(root)
	raw, err := os.ReadFile(chestLedger)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "OA-CHEST-"+archivistMissionID)
}

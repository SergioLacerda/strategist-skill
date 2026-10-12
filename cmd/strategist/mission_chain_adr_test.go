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
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	catalogadapter "github.com/SergioLacerda/strategist-skill/internal/runtime/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// adrChainResult is a writing-plans completion whose tasks.md is analysis-only
// and declares one approved OA-ADR side quest (R13): the package carries no
// implementation target, so the only thing that can reach the Sniper is the
// ADR the operator accepts at the Approval Gate.
const adrChainResult = `---
handoff_policy_facts:
  schema_version: strategist-handoff-policy-facts/v1
  mandatory_constraints: []
  unresolved_questions: []
  forbidden_scope: []
  destructive_operation_possible: false
  security_sensitive_task: false
  informational_only: true
---

## proposal.md

### Why

reason

## design.md

### Context

context

## tasks.md

- [ ] 1.1 [task_type: analysis_artifact] record the evidence and verify the note exists
` + "```yaml\nside_quests_approved:\n  - {id: OA-ADR-" + archivistMissionID + ", description: record the decision, strategy: execute_together, status: sq_pending}\n```\n"

// chainWorkspace is the Archivist workspace (refinement slot bound to the
// embedded writing-plans Weapon, Ranger outcome authorized) plus the embedded
// Sniper binding, with the mission advanced through the real FSM to refinement.
func chainWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root, basePath := archivistWorkspace(t, true)
	catalog, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	registry, err := catalogadapter.ParseCompiledRegistryCatalog(catalog)
	require.NoError(t, err)
	offers := registry.RankedBindingsFor("sniper", "execution")
	require.Len(t, offers, 1)
	offer := offers[0]
	lock := readChainYAML[domain.PluginLockFile](t, filepath.Join(root, "plugins.lock"))
	lock.Bindings = append(lock.Bindings, domain.SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: "execution", InstalledInstanceID: offer.WeaponID,
		Role: "sniper", WeaponVersion: offer.WeaponVersion, WeaponDigest: offer.WeaponDigest, SourceDigest: offer.SourceDigest,
		BindingDigest: offer.BindingDigest, ExecutionMode: offer.ExecutionMode, Origin: string(domain.WeaponOriginEmbedded),
		RuntimeKind: offer.Runtime.Kind, ConnectorID: offer.ConnectorID, Entrypoint: offer.Entrypoint,
		CertificationDigest: offer.CertificationDigest, Status: offer.Status, Mode: domain.SlotBindingModeRanked,
	})
	writeYAML(t, filepath.Join(root, "plugins.lock"), lock)
	active := readChainYAML[domain.ActiveConfig](t, filepath.Join(root, "active.yaml"))
	active.Slots["execution"] = offer.WeaponID
	active.BasePath = ".analysis"
	writeYAML(t, filepath.Join(root, "active.yaml"), active)
	advanceChainMission(t, root, domain.MissionEventDiscoveryDone)
	return root, basePath
}

// advanceChainMission replaces the bare status the Archivist fixture wrote with
// an engine that walked the real transitions up to the given event.
func advanceChainMission(t *testing.T, root string, last domain.MissionEngineEvent) {
	t.Helper()
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: archivistMissionID})
	require.NoError(t, err)
	var status domain.MissionEngineStatus
	for _, event := range []domain.MissionEngineEvent{domain.MissionEventBootstrapDone, domain.MissionEventIntakeDone, last} {
		status, err = engine.Submit(event)
		require.NoError(t, err, event)
	}
	require.NoError(t, saveMission(root, status))
}

func chainSubmit(t *testing.T, root string, event domain.MissionEngineEvent) error {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return missionadapter.RunSubmit(cmd, missionLifecycleDependencies(), root, archivistMissionID, string(event), false)
}

func chainAcceptADR(t *testing.T, root string) {
	t.Helper()
	cmd := missionadapter.NewAcceptSideQuest(missionLifecycleDependencies())
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--root", root, "--mission-id", archivistMissionID, "--side-quest", "OA-ADR-" + archivistMissionID})
	require.NoError(t, cmd.Execute())
}

func chainState(t *testing.T, root string) domain.MissionState {
	t.Helper()
	_, status, err := loadMission(root, archivistMissionID)
	require.NoError(t, err)
	return status.State
}

// R13: the whole chain without a hand-written snapshot. The Archivist publishes
// an analysis-only package carrying an OA-ADR; the operator accepts it at the
// gate; the handoff passes; execution entry derives the ADR target from the
// sealed record; the Sniper request carries exactly that target; and the
// Sniper completion verifies only because the ledger proves its materialization.
func TestWritingPlansOAADRChainReachesSniperClosureThroughTheDerivedTarget(t *testing.T) {
	root, basePath := chainWorkspace(t)
	request, err := buildMissionInvocation(t.Context(), archivistBuildInput(root, basePath))
	require.NoError(t, err)
	_, err = completeMissionInvocation(t.Context(), archivistComplete(root, basePath, request.RequestID, adrChainResult))
	require.NoError(t, err)

	require.NoError(t, chainSubmit(t, root, domain.MissionEventRefinementDone))
	require.Equal(t, domain.StateApprovalGate, chainState(t, root))
	chainAcceptADR(t, root)
	require.NoError(t, chainSubmit(t, root, domain.MissionEventGateApproved))
	require.Equal(t, domain.StateHandoffChallenge, chainState(t, root))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runArchivistHandoff(cmd, handoffEvaluateOptions{Root: root, MissionID: archivistMissionID}))
	require.NoError(t, chainSubmit(t, root, domain.MissionEventHandoffSatisfied))
	require.Equal(t, domain.StateExecution, chainState(t, root))

	sniper, err := buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{
		Root: root, BasePath: basePath, MissionID: archivistMissionID, Role: "sniper", Slot: "execution",
	})
	require.NoError(t, err)
	targets, ok := sniper.Input["execution_targets"].([]domain.ExecutionTarget)
	require.True(t, ok, "the Sniper request carries typed execution targets")
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, domain.ExecutionTargetKindADR, target.Kind)
	assert.Equal(t, domain.ExecutionTargetOriginAcceptedSideQuest, target.Origin)
	assert.Equal(t, "OA-ADR-"+archivistMissionID, target.SideQuestID)
	assert.Equal(t, ".analysis/archived/"+archivistMissionID+"-adr.md", target.Path, "no canonical ADR root is configured: the decision stays beside the report")

	closeSniperWithoutProof := func() error {
		_, err := completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
			Root: root, BasePath: basePath, RequestID: sniper.RequestID, Adapter: domain.ExecutionAdapterCurrentHost,
			Completion: domain.MissionInvocationCompletion{RequestID: sniper.RequestID, Result: sniperChainResult},
		})
		return err
	}
	materializeChainReport(t, basePath)
	require.ErrorContains(t, closeSniperWithoutProof(), "has no materialization ledger proof", "the ADR was not materialized")

	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), target.Path), []byte("# ADR\n"), 0o600))
	require.NoError(t, telemetry.AppendSniperMaterialization(telemetry.SniperMaterializationHistoryPath(root), telemetry.SniperMaterializationRecord{
		MissionID: archivistMissionID, BasePath: basePath, TargetPath: target.Path, MaterializedAt: time.Now().UTC(),
	}))
	require.NoError(t, closeSniperWithoutProof())
	accepted, found, err := livemission.LoadAcceptedSideQuest(root, archivistMissionID)
	require.NoError(t, err)
	require.True(t, found)
	assert.True(t, accepted.Sealed)
}

const sniperChainResult = "sniper: done | report_path: .analysis/archived/arch-mission-report.md | mission_status: documentation_applied"

// materializeChainReport writes the Sniper's report and marks the package the
// way the Sniper does, leaving the target ledger as the only missing proof.
func materializeChainReport(t *testing.T, basePath string) {
	t.Helper()
	refined := filepath.Join(basePath, "refined", archivistMissionID)
	analysis := filepath.Join(refined, "analysis.md")
	raw, err := os.ReadFile(analysis)
	require.NoError(t, err)
	updated := strings.Replace(string(raw), "mission_status: gate_analysis_accepted", "mission_status: documentation_applied\nclaimed_by: test-session", 1)
	require.NotEqual(t, string(raw), updated, "the gate accepted the analysis before execution")
	require.NoError(t, os.WriteFile(analysis, []byte(updated), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "archived"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "archived", archivistMissionID+"-report.md"), []byte("report\n"), 0o600))
}

func readChainYAML[T any](t *testing.T, path string) T {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var value T
	require.NoError(t, yaml.Unmarshal(raw, &value))
	return value
}

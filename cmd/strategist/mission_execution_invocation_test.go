package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	catalogadapter "github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestSniperCurrentHostInvocationCompletesWithMaterializationProof(t *testing.T) {
	root, basePath := sniperInvocationWorkspace(t)
	request, err := buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{
		Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: "execution",
	})
	require.NoError(t, err)
	require.Contains(t, request.Payload, "# Sniper")
	require.NotEmpty(t, request.SourceDigest)
	require.Contains(t, request.Input["execution_contract"], "current-host")

	refined := filepath.Join(basePath, "refined", flowMissionID)
	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("---\nmission_id: "+flowMissionID+"\nmission_status: documentation_applied\nclaimed_by: test-session\n---\n\nbody\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(refined, "tasks.md"), []byte("- [x] 1.1 [documentation_target] Write `docs/guide.md`.\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(filepath.Dir(root), "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), "docs", "guide.md"), []byte("guide\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "archived"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "archived", flowMissionID+"-report.md"), []byte("report\n"), 0o600))
	require.NoError(t, telemetry.AppendSniperMaterialization(telemetry.SniperMaterializationHistoryPath(root), telemetry.SniperMaterializationRecord{
		MissionID: flowMissionID, BasePath: basePath, TargetPath: "docs/guide.md", MaterializedAt: time.Now().UTC(),
	}))

	outcome, err := completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
		Root: root, BasePath: basePath, RequestID: request.RequestID, Adapter: domain.ExecutionAdapterCurrentHost,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: "sniper: done | report_path: .analysis/archived/flow-mission-report.md | mission_status: documentation_applied"},
	})
	require.NoError(t, err)
	require.Equal(t, "verified", outcome.Status)
	require.Equal(t, ".analysis/archived/flow-mission-report.md", outcome.ArtifactPath)

	_, err = completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{Root: root, BasePath: basePath, RequestID: request.RequestID})
	require.ErrorContains(t, err, "invocation_replay")
}

func sniperInvocationWorkspace(t *testing.T) (string, string) {
	t.Helper()
	workspace := t.TempDir()
	root := filepath.Join(workspace, ".strategist")
	basePath := filepath.Join(workspace, ".analysis")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	catalog, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), catalog, 0o600))
	registry, err := catalogadapter.ParseCompiledRegistryCatalog(catalog)
	require.NoError(t, err)
	offers := registry.RankedBindingsFor("sniper", "execution")
	require.Len(t, offers, 1)
	offer := offers[0]
	binding := domain.SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: "execution", InstalledInstanceID: offer.WeaponID,
		Role: "sniper", WeaponVersion: offer.WeaponVersion, WeaponDigest: offer.WeaponDigest, SourceDigest: offer.SourceDigest,
		BindingDigest: offer.BindingDigest, ExecutionMode: offer.ExecutionMode, Origin: string(domain.WeaponOriginEmbedded),
		RuntimeKind: offer.Runtime.Kind, ConnectorID: offer.ConnectorID, Entrypoint: offer.Entrypoint,
		CertificationDigest: offer.CertificationDigest, Status: offer.Status, Mode: domain.SlotBindingModeRanked,
	}
	writeYAML(t, filepath.Join(root, "active.yaml"), domain.ActiveConfig{Slots: map[string]string{"execution": offer.WeaponID}})
	writeYAML(t, filepath.Join(root, "plugins.lock"), domain.PluginLockFile{Bindings: []domain.SlotBinding{binding}})
	refined := filepath.Join(basePath, "refined", flowMissionID)
	require.NoError(t, os.MkdirAll(refined, 0o755))
	for name, content := range map[string]string{
		"analysis.md": "---\nmission_id: " + flowMissionID + "\nmission_status: gate_analysis_accepted\n---\n\nbody\n",
		"proposal.md": "proposal\n", "design.md": "design\n",
		"tasks.md": "- [ ] 1.1 [documentation_target] Write `docs/guide.md`.\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(refined, name), []byte(content), 0o600))
	}
	digest, err := handoff.PackageDigest(refined)
	require.NoError(t, err)
	status, err := json.Marshal(domain.MissionEngineStatus{MissionID: flowMissionID, Phase: domain.PhaseExecution, State: domain.StateExecution, ApprovalGatePackageDigest: digest})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", flowMissionID+".json"), status, 0o600))
	return root, basePath
}

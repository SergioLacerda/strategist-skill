package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const flowMissionID = "flow-mission"

// rankedWorkspace builds a hermetic .strategist root whose discovery slot is
// bound to the certified Ranked Embedded brainstorming Weapon.
func rankedWorkspace(t *testing.T, mutate func(*domain.SlotBinding)) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))

	catalog, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), catalog, 0o600))
	registry, err := domain.ParseCompiledRegistryCatalog(catalog)
	require.NoError(t, err)
	ranked := registry.RankedBindingsFor("ranger", "discovery")
	require.NotEmpty(t, ranked)
	offer := ranked[0]

	binding := domain.SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", Slot: "discovery", InstalledInstanceID: offer.WeaponID,
		Role: "ranger", WeaponVersion: offer.WeaponVersion, WeaponDigest: offer.WeaponDigest, SourceDigest: offer.SourceDigest,
		BindingDigest: offer.BindingDigest, ExecutionMode: offer.ExecutionMode, Origin: string(domain.WeaponOriginEmbedded),
		RuntimeKind: offer.Runtime.Kind, ConnectorID: offer.ConnectorID, Entrypoint: offer.Entrypoint,
		CertificationDigest: offer.CertificationDigest, Status: offer.Status, Mode: domain.SlotBindingModeRanked,
	}
	if mutate != nil {
		mutate(&binding)
	}
	writeYAML(t, filepath.Join(root, "active.yaml"), domain.ActiveConfig{Slots: map[string]string{"discovery": offer.WeaponID}})
	writeYAML(t, filepath.Join(root, "plugins.lock"), domain.PluginLockFile{Bindings: []domain.SlotBinding{binding}})

	status, err := json.Marshal(domain.MissionEngineStatus{MissionID: flowMissionID, Phase: domain.PhaseDiscovery})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", flowMissionID+".json"), status, 0o600))
	return root
}

func writeYAML(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := yaml.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

func flowBuildInput(root string) missionadapter.InvocationBuildInput {
	return missionadapter.InvocationBuildInput{Root: root, BasePath: filepath.Join(filepath.Dir(root), ".analysis"), MissionID: flowMissionID, Role: "ranger", Slot: "discovery", RequestContext: "evaluate"}
}

func TestBuildMissionInvocationIssuesARankedEmbeddedRequest(t *testing.T) {
	root := rankedWorkspace(t, nil)

	request, err := buildMissionInvocation(t.Context(), flowBuildInput(root))

	require.NoError(t, err)
	require.NotEmpty(t, request.Payload)
	require.Equal(t, "brainstorming", request.Weapon.ID)
	require.Contains(t, request.Input["output_contract"], "## mission_objective")
	require.Equal(t, "evaluate", request.Input["request_context"])
}

func TestBuildMissionInvocationRejectsALockThatDriftedFromTheCompiledBinding(t *testing.T) {
	root := rankedWorkspace(t, func(b *domain.SlotBinding) { b.BindingDigest = "sha256:tampered" })

	_, err := buildMissionInvocation(t.Context(), flowBuildInput(root))

	require.ErrorContains(t, err, "does not match compiled binding")
}

func TestBuildMissionInvocationRefusesACustomBinding(t *testing.T) {
	root := rankedWorkspace(t, func(b *domain.SlotBinding) { b.Mode = domain.SlotBindingModeCustom })

	_, err := buildMissionInvocation(t.Context(), flowBuildInput(root))

	require.ErrorContains(t, err, "only supports Ranked")
}

func TestBuildMissionInvocationRejectsAWorkspaceCatalogThatDiffersFromTheBinary(t *testing.T) {
	root := rankedWorkspace(t, nil)
	catalogPath := filepath.Join(root, "plugins", "catalog.yaml")
	raw, err := os.ReadFile(catalogPath)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	doc["ranked_bindings"] = []any{}
	writeYAML(t, catalogPath, doc)

	_, err = buildMissionInvocation(t.Context(), flowBuildInput(root))

	require.ErrorContains(t, err, "compiled_registry_drift")
}

const flowResult = "## mission_objective\nx\n## known_facts\n- f\n## confidence_summary\nc\n## handoff\nh\n"

func TestInvokeHostCompleteEndToEndThenReplayIsRejected(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	complete := missionadapter.InvocationCompleteInput{
		Root: root, BasePath: input.BasePath, RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: flowResult},
	}

	outcome, err := completeMissionInvocation(t.Context(), complete)

	require.NoError(t, err)
	require.Equal(t, "normalized", outcome.Status)
	written, err := os.ReadFile(filepath.Join(filepath.Dir(root), outcome.ArtifactPath))
	require.NoError(t, err)
	require.Contains(t, string(written), "sources_consulted: []")
	require.Contains(t, string(written), "mission_status: ranger_pending")

	_, err = completeMissionInvocation(t.Context(), complete)
	require.ErrorContains(t, err, "invocation_replay")
}

func TestCompleteRejectsACompletionAfterTheBindingChanged(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	rankedWorkspaceRebind(t, root, "sha256:changed")

	_, err = completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
		Root: root, BasePath: input.BasePath, RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: flowResult},
	})

	require.Error(t, err)
}

func rankedWorkspaceRebind(t *testing.T, root, digest string) {
	t.Helper()
	path := filepath.Join(root, "plugins.lock")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var lock domain.PluginLockFile
	require.NoError(t, yaml.Unmarshal(raw, &lock))
	lock.Bindings[0].BindingDigest = digest
	writeYAML(t, path, lock)
}

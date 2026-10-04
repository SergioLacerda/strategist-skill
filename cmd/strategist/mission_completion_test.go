package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/require"
)

const completionRequestID = "inv_0123456789abcdef"

func completionFixture(t *testing.T, phase domain.PipelinePhase) (string, missionadapter.InvocationCompleteInput) {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	request := domain.MissionInvocationRequest{
		Protocol: domain.MissionInvocationProtocolVersion, RequestID: completionRequestID, MissionID: "m1",
		Role: "ranger", Slot: string(domain.SlotDiscovery),
		Weapon:        domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:w"},
		BindingDigest: "sha256:b", SourceDigest: "sha256:s", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "p",
	}
	now := time.Now().UTC()
	require.NoError(t, missionruntime.NewInvocationStore(root).Put(domain.MissionInvocationRecord{Request: request, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	raw, err := json.Marshal(domain.MissionEngineStatus{MissionID: "m1", Phase: phase})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", "m1.json"), raw, 0o600))
	return root, missionadapter.InvocationCompleteInput{
		Root: root, BasePath: filepath.Join(filepath.Dir(root), ".analysis"), RequestID: completionRequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: completionRequestID, Result: "body"},
		Adapter:    domain.ExecutionAdapterCurrentHost,
		Sink:       &captureSink{},
	}
}

func TestCompleteMissionInvocationRejectsAConcurrentCompletion(t *testing.T) {
	root, input := completionFixture(t, domain.PhaseDiscovery)
	_, err := missionruntime.NewInvocationStore(root).ClaimTarget("m1", "ranger", string(domain.SlotDiscovery))
	require.NoError(t, err)

	_, err = completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_in_progress")
}

func TestCompleteMissionInvocationRejectsAMissionOutsideDiscovery(t *testing.T) {
	_, input := completionFixture(t, domain.PhaseRefinement)

	_, err := completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_phase_mismatch")
}

func TestCompleteMissionInvocationRejectsAnotherRequestsCompletion(t *testing.T) {
	_, input := completionFixture(t, domain.PhaseDiscovery)
	input.Completion.RequestID = "inv_fedcba9876543210"

	_, err := completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_binding_mismatch")
}

func TestCompleteMissionInvocationRejectsAnEmptyResult(t *testing.T) {
	_, input := completionFixture(t, domain.PhaseDiscovery)
	input.Completion.Result = "  "

	_, err := completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "result is required")
}

func TestCompleteMissionInvocationRejectsAnExpiredRequest(t *testing.T) {
	root, input := completionFixture(t, domain.PhaseDiscovery)
	path := filepath.Join(root, "missions", "invocations", completionRequestID+".json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var record domain.MissionInvocationRecord
	require.NoError(t, json.Unmarshal(raw, &record))
	record.ExpiresAt = time.Now().Add(-time.Minute)
	raw, err = json.Marshal(record)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	_, err = completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_request_expired")
}

func TestDiscoveryArtifactPathsRejectsWorkspaceEscape(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws", ".strategist")
	_, _, err := discoveryArtifactPaths(root, filepath.Join(root, "..", "..", "outside"), "m1")
	require.ErrorContains(t, err, "escapes workspace")
}

func TestRequireRegistryMatchesBinary(t *testing.T) {
	raw, err := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	registry, err := catalog.ParseCompiledRegistryCatalog(raw)
	require.NoError(t, err)

	require.NoError(t, requireRegistryMatchesBinary(registry))

	registry.RankedBindings = registry.RankedBindings[:len(registry.RankedBindings)-1]
	require.ErrorContains(t, requireRegistryMatchesBinary(registry), "compiled_registry_drift")
}

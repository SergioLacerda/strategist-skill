package mission

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestInvokeEmitsBuildRequest(t *testing.T) {
	request := domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
	deps := invocationTestDependencies(request)
	cmd := NewInvoke(deps)
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--mission-id", "mission", "--slot", "discovery", "--json"})
	require.NoError(t, cmd.Execute())
	var got domain.MissionInvocationRequest
	require.NoError(t, json.Unmarshal(output.Bytes(), &got))
	require.Equal(t, request, got)
}

func TestInvokeRequiresTheSlotPhase(t *testing.T) {
	request := domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
	deps := invocationTestDependencies(request)
	deps.LoadMission = func(string, string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{MissionID: "mission", Phase: domain.PhaseBootstrap, State: domain.StateInit}, nil
	}
	cmd := NewInvoke(deps)
	cmd.SetArgs([]string{"--mission-id", "mission", "--slot", "discovery", "--json"})

	err := cmd.Execute()
	require.ErrorContains(t, err, `slot "discovery" requires phase "DISCOVERY", got "BOOTSTRAP"`)
}

func TestValidateInvocationPhase(t *testing.T) {
	require.NoError(t, validateInvocationPhase(domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, "discovery"))
	require.NoError(t, validateInvocationPhase(domain.MissionEngineStatus{Phase: domain.PhaseRefinement}, "refinement"))
	require.ErrorContains(t, validateInvocationPhase(domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, "execution"), "has no host invocation boundary")
}

func TestCompleteReadsOneRawResponse(t *testing.T) {
	deps := invocationTestDependencies(domain.MissionInvocationRequest{})
	deps.ReadCompletion = func(*cobra.Command) (domain.MissionInvocationCompletion, error) {
		return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw result"}, nil
	}
	deps.Complete = func(context.Context, InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
		return domain.MissionInvocationOutcome{RequestID: "inv_12345678", Status: "normalized"}, nil
	}
	cmd := NewComplete(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--request-id", "inv_12345678", "--json"})
	require.NoError(t, cmd.Execute())
}

func TestInvokeExecutesExplicitHostBridge(t *testing.T) {
	request := domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
	deps := invocationTestDependencies(request)
	called := false
	deps.ExecuteHost = func(_ context.Context, root, host, requestContext string, got domain.MissionInvocationRequest) (domain.MissionInvocationCompletion, error) {
		called = true
		require.Equal(t, "root", root)
		require.Equal(t, "codex", host)
		require.Equal(t, "analyze this", requestContext)
		require.Equal(t, request, got)
		return domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: "raw result"}, nil
	}
	cmd := NewInvoke(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--root", "root", "--mission-id", "mission", "--slot", "discovery", "--host", "codex", "--context", "analyze this", "--json"})
	require.NoError(t, cmd.Execute())
	require.True(t, called)
}

func TestInvokePassesRequestContextToTheImmutableRequest(t *testing.T) {
	request := domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
	deps := invocationTestDependencies(request)
	deps.Build = func(_ context.Context, input InvocationBuildInput) (domain.MissionInvocationRequest, error) {
		require.Equal(t, "analyze this", input.RequestContext)
		return request, nil
	}
	cmd := NewInvoke(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--root", "root", "--mission-id", "mission", "--slot", "discovery", "--context", "analyze this", "--json"})
	require.NoError(t, cmd.Execute())
}

func TestInvokeHostBridgeRequiresContext(t *testing.T) {
	deps := invocationTestDependencies(domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"})
	cmd := NewInvoke(deps)
	cmd.SetArgs([]string{"--mission-id", "mission", "--slot", "discovery", "--host", "codex"})
	require.ErrorContains(t, cmd.Execute(), "--context is required with --host")
}

func invocationTestDependencies(request domain.MissionInvocationRequest) InvocationDependencies {
	return InvocationDependencies{
		RootFlag: "root", RequireMissionID: func(string) error { return nil },
		ResolveBasePath: func(root string) (string, string, error) { return root, ".analysis", nil },
		LoadMission: func(_ string, missionID string) (domain.MissionEngineStatus, error) {
			return domain.MissionEngineStatus{MissionID: missionID, Phase: domain.PhaseDiscovery, State: domain.StateInit}, nil
		},
		Build: func(context.Context, InvocationBuildInput) (domain.MissionInvocationRequest, error) {
			return request, nil
		},
		Complete: func(context.Context, InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
			return domain.MissionInvocationOutcome{Status: "normalized"}, nil
		},
		WriteResult: func(cmd *cobra.Command, asJSON bool, value any) error {
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
			}
			return nil
		},
		ReadCompletion: func(*cobra.Command) (domain.MissionInvocationCompletion, error) {
			return domain.MissionInvocationCompletion{}, nil
		},
	}
}

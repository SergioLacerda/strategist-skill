package mission

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
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

func adapterTestRequest() domain.MissionInvocationRequest {
	return domain.MissionInvocationRequest{Protocol: domain.MissionInvocationProtocolVersion, RequestID: "inv_12345678", MissionID: "mission", Role: "ranger", Slot: "discovery", Weapon: domain.MissionWeaponIdentity{ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon"}, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", ExecutionMode: "prompt_bridge", Entrypoint: "discover", Payload: "payload"}
}

func TestInvokeHostCompletesAsTheMatchingChildAdapter(t *testing.T) {
	for host, want := range map[string]domain.MissionExecutionAdapter{"codex": domain.ExecutionAdapterCodexChild, "claude": domain.ExecutionAdapterClaudeChild} {
		deps := invocationTestDependencies(adapterTestRequest())
		deps.ExecuteHost = func(context.Context, string, string, string, domain.MissionInvocationRequest) (domain.MissionInvocationCompletion, error) {
			return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw"}, nil
		}
		var got domain.MissionExecutionAdapter
		deps.Complete = func(_ context.Context, input InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
			got = input.Adapter
			return domain.MissionInvocationOutcome{Status: "normalized"}, nil
		}
		cmd := NewInvoke(deps)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetArgs([]string{"--root", "r", "--mission-id", "mission", "--slot", "discovery", "--host", host, "--context", "c", "--json"})

		require.NoError(t, cmd.Execute())
		require.Equal(t, want, got, host)
	}
}

func TestInvokeRejectsAnUnknownHostBeforeAnyChildRuns(t *testing.T) {
	deps := invocationTestDependencies(adapterTestRequest())
	deps.ExecuteHost = func(context.Context, string, string, string, domain.MissionInvocationRequest) (domain.MissionInvocationCompletion, error) {
		t.Fatal("no child may launch for an unknown host")
		return domain.MissionInvocationCompletion{}, nil
	}
	deps.Complete = func(context.Context, InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
		t.Fatal("no completion may be attempted")
		return domain.MissionInvocationOutcome{}, nil
	}
	cmd := NewInvoke(deps)
	cmd.SetArgs([]string{"--root", "r", "--mission-id", "mission", "--slot", "discovery", "--host", "gemini", "--context", "c"})

	require.ErrorContains(t, cmd.Execute(), "invocation_adapter_unknown")
}

func TestCompleteAlwaysDeclaresTheCurrentHostAdapter(t *testing.T) {
	deps := invocationTestDependencies(domain.MissionInvocationRequest{})
	deps.ReadCompletion = func(*cobra.Command) (domain.MissionInvocationCompletion, error) {
		return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw"}, nil
	}
	var got domain.MissionExecutionAdapter
	deps.Complete = func(_ context.Context, input InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
		got = input.Adapter
		return domain.MissionInvocationOutcome{Status: "normalized"}, nil
	}
	cmd := NewComplete(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--request-id", "inv_12345678", "--json"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, domain.ExecutionAdapterCurrentHost, got)
}

type markerSink struct{ name string }

func (markerSink) Emit(context.Context, telemetry.Event) error { return nil }

func TestCompletionPathsReceiveTheInjectedTelemetrySink(t *testing.T) {
	want := markerSink{name: "injected"}
	build := func() InvocationDependencies {
		deps := invocationTestDependencies(adapterTestRequest())
		deps.TelemetrySink = func() telemetry.EventSink { return want }
		deps.ReadCompletion = func(*cobra.Command) (domain.MissionInvocationCompletion, error) {
			return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw"}, nil
		}
		deps.ExecuteHost = func(context.Context, string, string, string, domain.MissionInvocationRequest) (domain.MissionInvocationCompletion, error) {
			return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw"}, nil
		}
		return deps
	}
	for name, run := range map[string]func(InvocationDependencies) *cobra.Command{
		"complete": func(d InvocationDependencies) *cobra.Command {
			c := NewComplete(d)
			c.SetArgs([]string{"--request-id", "inv_12345678", "--json"})
			return c
		},
		"invoke host": func(d InvocationDependencies) *cobra.Command {
			c := NewInvoke(d)
			c.SetArgs([]string{"--root", "r", "--mission-id", "mission", "--slot", "discovery", "--host", "claude", "--context", "c", "--json"})
			return c
		},
	} {
		deps := build()
		var got telemetry.EventSink
		deps.Complete = func(_ context.Context, input InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
			got = input.Sink
			return domain.MissionInvocationOutcome{Status: "normalized"}, nil
		}
		cmd := run(deps)
		cmd.SetOut(&bytes.Buffer{})

		require.NoError(t, cmd.Execute(), name)
		require.Equal(t, want, got, name)
	}
}

func TestMissingTelemetrySelectorYieldsNoSinkForTheRuntimeToRefuse(t *testing.T) {
	deps := invocationTestDependencies(adapterTestRequest())
	var got telemetry.EventSink = markerSink{}
	deps.Complete = func(_ context.Context, input InvocationCompleteInput) (domain.MissionInvocationOutcome, error) {
		got = input.Sink
		return domain.MissionInvocationOutcome{}, nil
	}
	deps.ReadCompletion = func(*cobra.Command) (domain.MissionInvocationCompletion, error) {
		return domain.MissionInvocationCompletion{RequestID: "inv_12345678", Result: "raw"}, nil
	}
	cmd := NewComplete(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--request-id", "inv_12345678"})

	require.NoError(t, cmd.Execute())
	require.Nil(t, got)
}

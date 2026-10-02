package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMissionInvocationFailureModes(t *testing.T) {
	root := rankedWorkspace(t, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := buildMissionInvocation(ctx, flowBuildInput(root))
	require.ErrorContains(t, err, "context")

	_, err = buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{Root: t.TempDir(), MissionID: flowMissionID, Role: "ranger", Slot: "discovery"})
	require.ErrorContains(t, err, "read active.yaml")

	unbound := flowBuildInput(root)
	unbound.Role, unbound.Slot = "archivist", "refinement"
	_, err = buildMissionInvocation(t.Context(), unbound)
	require.ErrorContains(t, err, "role_invocation_failed")

	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", "invocations"), nil, 0o644))
	_, err = buildMissionInvocation(t.Context(), flowBuildInput(root))
	require.ErrorContains(t, err, "persist mission invocation")
}

func TestBuildMissionInvocationOmitsEmptyRequestContext(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	input.RequestContext = "  "
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	assert.NotContains(t, request.Input, "request_context")
	assert.Contains(t, request.Input, "output_contract")

	id, err := newInvocationID()
	require.NoError(t, err)
	assert.Contains(t, id, "inv_")
}

func TestCompleteMissionInvocationFailureModes(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	base := missionadapter.InvocationCompleteInput{
		Root: root, BasePath: input.BasePath, RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: flowResult},
		Adapter:    domain.ExecutionAdapterCurrentHost, Sink: &captureSink{},
	}

	noSink := base
	noSink.Sink = nil
	_, err = completeMissionInvocation(t.Context(), noSink)
	require.ErrorContains(t, err, "invocation_telemetry_unavailable")

	unknown := base
	unknown.RequestID = "inv_unknown"
	_, err = completeMissionInvocation(t.Context(), unknown)
	require.ErrorContains(t, err, "load mission invocation")

	mismatched := base
	mismatched.Completion.RequestID = "inv_other"
	_, err = completeMissionInvocation(t.Context(), mismatched)
	require.Error(t, err)

	empty := base
	empty.Completion.Result = ""
	_, err = completeMissionInvocation(t.Context(), empty)
	require.Error(t, err)

	require.NoError(t, os.Remove(filepath.Join(root, "missions", flowMissionID+".json")))
	_, err = completeMissionInvocation(t.Context(), base)
	require.ErrorContains(t, err, "invocation_phase_mismatch")
}

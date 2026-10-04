package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCompleteInvocationDelegatesRequest(t *testing.T) {
	t.Parallel()

	called := false
	want := domain.MissionInvocationOutcome{RequestID: "req-1"}
	got, err := application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{RequestID: "req-1"}, func(_ context.Context, request application.InvocationCompletionRequest) (domain.MissionInvocationOutcome, error) {
		called = true
		require.Equal(t, "req-1", request.RequestID)
		return want, nil
	})

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, want, got)
}

func TestCompleteInvocationPropagatesRuntimeError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("normalization failed")
	_, err := application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{RequestID: "req-1"}, func(context.Context, application.InvocationCompletionRequest) (domain.MissionInvocationOutcome, error) {
		return domain.MissionInvocationOutcome{}, wantErr
	})
	require.ErrorIs(t, err, wantErr)
}

func TestCompleteInvocationRejectsMissingRequestIDAndPort(t *testing.T) {
	t.Parallel()

	_, err := application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{}, nil)
	require.EqualError(t, err, "mission complete: invocation runtime is unavailable")

	_, err = application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{}, func(context.Context, application.InvocationCompletionRequest) (domain.MissionInvocationOutcome, error) {
		return domain.MissionInvocationOutcome{}, nil
	})
	require.EqualError(t, err, "--request-id is required")
}

func TestValidateInvocationDispatchRejectsUnsafeChildHostRequests(t *testing.T) {
	t.Parallel()

	require.NoError(t, application.ValidateInvocationDispatch("", "", ""))
	require.EqualError(t, application.ValidateInvocationDispatch("codex", "", "discovery"), "--context is required with --host")
	require.ErrorContains(t, application.ValidateInvocationDispatch("codex", "request", string(domain.SlotExecution)), "execution requires the current-host adapter")
	require.ErrorContains(t, application.ValidateInvocationDispatch("unknown", "request", "discovery"), "invocation_adapter_unknown")
}

func TestValidateInvocationPhaseMapsSlotsToPipelinePhases(t *testing.T) {
	t.Parallel()

	require.NoError(t, application.ValidateInvocationPhase(domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, string(domain.SlotDiscovery)))
	require.ErrorContains(t, application.ValidateInvocationPhase(domain.MissionEngineStatus{Phase: domain.PhaseExecution}, string(domain.SlotDiscovery)), "requires phase")
	require.ErrorContains(t, application.ValidateInvocationPhase(domain.MissionEngineStatus{}, "unknown"), "no host invocation boundary")
}

func TestPrepareInvocationLoadsBeforeValidatingPhase(t *testing.T) {
	t.Parallel()

	loaded := false
	err := application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), func(_, _ string) (domain.MissionEngineStatus, error) {
		loaded = true
		return domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, nil
	})

	require.NoError(t, err)
	require.True(t, loaded)
}

func TestPrepareInvocationPropagatesLoadAndMissingPortErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("load failed")
	err := application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), func(_, _ string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{}, wantErr
	})
	require.ErrorIs(t, err, wantErr)
	require.EqualError(t, application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), nil), "mission lifecycle is unavailable")
}

func TestBuildInvocationDelegatesImmutableRequest(t *testing.T) {
	t.Parallel()

	want := domain.MissionInvocationRequest{RequestID: "req-1"}
	got, err := application.BuildInvocation(context.Background(), application.InvocationBuildRequest{MissionID: "m-1", Slot: "discovery"}, func(_ context.Context, request application.InvocationBuildRequest) (domain.MissionInvocationRequest, error) {
		require.Equal(t, "m-1", request.MissionID)
		return want, nil
	})

	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestBuildInvocationRejectsMissingBuilder(t *testing.T) {
	t.Parallel()

	_, err := application.BuildInvocation(context.Background(), application.InvocationBuildRequest{}, nil)
	require.EqualError(t, err, "mission invoke: invocation runtime is unavailable")
}

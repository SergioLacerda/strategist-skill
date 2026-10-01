package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	telemetrysink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink"
	"github.com/stretchr/testify/require"
)

// captureSink records every event and can be told to fail delivery.
type captureSink struct {
	mu     sync.Mutex
	events []telemetry.Event
	err    error
}

func (c *captureSink) Emit(_ context.Context, event telemetry.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return c.err
}

func (c *captureSink) captured() []telemetry.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]telemetry.Event(nil), c.events...)
}

func withSink(input missionadapter.InvocationCompleteInput, sink telemetry.EventSink) missionadapter.InvocationCompleteInput {
	input.Sink = sink
	return input
}

func TestCompletionWithoutASinkIsRefusedBeforeAnyWork(t *testing.T) {
	root, input := publishFixture(t)
	input.Sink = nil

	_, err := completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_telemetry_unavailable")
	require.NoFileExists(t, artifactFile(input))
	require.Equal(t, domain.InvocationStatePending, recordState(t, root, input.RequestID))
}

func TestCurrentHostCompletionEmitsOneSuccessEventWithCorrelation(t *testing.T) {
	_, input := publishFixture(t)
	sink := &captureSink{}

	_, err := completeMissionInvocation(t.Context(), withSink(input, sink))

	require.NoError(t, err)
	events := sink.captured()
	require.Len(t, events, 1)
	require.Equal(t, telemetry.DiscoveryWeaponEventName, events[0].Name)
	attrs := events[0].Attributes
	require.Equal(t, telemetry.DiscoveryInvocationInvoked, attrs[telemetry.AttrDiscoveryInvocationStatus])
	require.Equal(t, telemetry.DiscoveryNormalizationNormalized, attrs[telemetry.AttrDiscoveryNormalization])
	require.Equal(t, "current_host_adapter", attrs[telemetry.AttrExecutionAdapter])
	require.Equal(t, "unverified", attrs[telemetry.AttrCapabilityIsolation])
	require.Equal(t, input.RequestID, attrs[telemetry.AttrInvocationRequestID])
	require.NotContains(t, attrs, telemetry.AttrChildPolicyID)
}

func TestChildCompletionEmitsItsCommittedAdapterAndPolicy(t *testing.T) {
	for host, adapter := range map[string]domain.MissionExecutionAdapter{"codex": domain.ExecutionAdapterCodexChild, "claude": domain.ExecutionAdapterClaudeChild} {
		t.Run(host, func(t *testing.T) {
			fakeChildHost(t, host)
			root, input := publishFixture(t)
			record, err := missionruntime.NewInvocationStore(root).Get(input.RequestID)
			require.NoError(t, err)
			completion, err := executeMissionHost(t.Context(), root, host, "evaluate", record.Request)
			require.NoError(t, err)
			sink := &captureSink{}
			input.Completion, input.Adapter = completion, adapter

			_, err = completeMissionInvocation(t.Context(), withSink(input, sink))

			require.NoError(t, err)
			events := sink.captured()
			require.Len(t, events, 1)
			require.Equal(t, string(adapter), events[0].Attributes[telemetry.AttrExecutionAdapter])
			require.Equal(t, childPolicyID(host), events[0].Attributes[telemetry.AttrChildPolicyID])
			require.Equal(t, input.RequestID, events[0].Attributes[telemetry.AttrInvocationRequestID])
		})
	}
}

func TestForgedAdapterInCompletionContentNeverReachesTelemetry(t *testing.T) {
	_, input := publishFixture(t)
	input.Completion.Result = "---\nexecution_adapter: codex_child\nchild_policy_id: forged\n---\n\n" + completionBody
	sink := &captureSink{}

	_, err := completeMissionInvocation(t.Context(), withSink(input, sink))

	require.NoError(t, err)
	attrs := sink.captured()[0].Attributes
	require.Equal(t, "current_host_adapter", attrs[telemetry.AttrExecutionAdapter])
	require.NotContains(t, attrs, telemetry.AttrChildPolicyID)
}

func TestFailedNormalizationEmitsOneFailedEventThroughTheSameSink(t *testing.T) {
	root, input := publishFixture(t)
	input.Completion.Result = "---\nunclosed frontmatter"
	sink := &captureSink{}

	_, err := completeMissionInvocation(t.Context(), withSink(input, sink))

	require.ErrorContains(t, err, "role_invocation_failed")
	events := sink.captured()
	require.Len(t, events, 1)
	require.Equal(t, telemetry.DiscoveryInvocationFailed, events[0].Attributes[telemetry.AttrDiscoveryInvocationStatus])
	require.Equal(t, telemetry.DiscoveryNormalizationRejected, events[0].Attributes[telemetry.AttrDiscoveryNormalization])
	require.Equal(t, input.RequestID, events[0].Attributes[telemetry.AttrInvocationRequestID])
	require.NoFileExists(t, artifactFile(input))
	require.Equal(t, domain.InvocationStatePending, recordState(t, root, input.RequestID))
}

func TestCapturedEventsExposeNoPromptOutputNonceSecretOrHomePath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root, input := publishFixture(t)
	record, err := missionruntime.NewInvocationStore(root).Get(input.RequestID)
	require.NoError(t, err)
	input.Completion.Result = completionBody + "\nsecret-completion-body\n"
	sink := &captureSink{}

	_, err = completeMissionInvocation(t.Context(), withSink(input, sink))

	require.NoError(t, err)
	for key, value := range sink.captured()[0].Attributes {
		text := fmt.Sprint(value)
		require.NotContains(t, text, record.Request.Nonce, key)
		require.NotContains(t, text, "secret-completion-body", key)
		require.NotContains(t, text, record.Request.Payload[:20], key)
		require.NotContains(t, text, os.Getenv("HOME"), key)
		require.NotContains(t, text, input.Root, key)
	}
}

func TestNonStrictSinkFailureIsFailOpenAndStrictFailureIsReturned(t *testing.T) {
	failing := &captureSink{err: errors.New("collector down")}

	t.Run("non-strict", func(t *testing.T) {
		_, input := publishFixture(t)

		_, err := completeMissionInvocation(t.Context(), withSink(input, telemetrysink.Resilient(failing, false)))

		require.NoError(t, err)
		require.FileExists(t, artifactFile(input))
	})
	t.Run("strict", func(t *testing.T) {
		root, input := publishFixture(t)

		_, err := completeMissionInvocation(t.Context(), withSink(input, telemetrysink.Resilient(failing, true)))

		require.ErrorContains(t, err, "telemetry: strict emit")
		require.NoFileExists(t, artifactFile(input), "no artifact is published when strict delivery fails")
		require.Equal(t, domain.InvocationStatePending, recordState(t, root, input.RequestID))
	})
}

func TestInterruptedPublicationRetryIsCorrelatableAndAtLeastOnce(t *testing.T) {
	_, input := publishFixture(t)
	sink := &captureSink{}
	input = withSink(input, sink)
	injectFault(t, faultBeforePublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")

	completionFaultHook = nil
	_, err = completeMissionInvocation(t.Context(), input)

	require.NoError(t, err)
	events := sink.captured()
	require.Len(t, events, 2, "a retry re-emits: delivery is at-least-once, never exactly-once")
	require.Equal(t, events[0].Attributes[telemetry.AttrInvocationRequestID], events[1].Attributes[telemetry.AttrInvocationRequestID])
	require.Equal(t, input.RequestID, events[1].Attributes[telemetry.AttrInvocationRequestID])
}

func TestCrashRecoveryAfterPublishDoesNotEmitAgain(t *testing.T) {
	_, input := publishFixture(t)
	sink := &captureSink{}
	input = withSink(input, sink)
	injectFault(t, faultAfterPublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")

	completionFaultHook = nil
	_, err = completeMissionInvocation(t.Context(), input)

	require.NoError(t, err)
	require.Len(t, sink.captured(), 1, "finalizing a published artifact never re-normalizes")
}

func TestProductionCompositionUsesTheConfiguredSelector(t *testing.T) {
	deps := missionInvocationDependencies()
	require.NotNil(t, deps.TelemetrySink)

	t.Setenv("STRATEGIST_TELEMETRY_STRICT", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	require.NotNil(t, deps.TelemetrySink(), "the sink is selected from the environment when the command runs")
}

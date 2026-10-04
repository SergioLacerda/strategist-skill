package policy

import (
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
)

var testBinding = integration.Binding{Provider: "jev", Version: "1", Model: "jev-1.13.0", Consumer: "handoff", Capability: integration.CapHandoffValidate}

func TestSelectEveryStateHasADeterministicPath(t *testing.T) {
	t.Run("ok selects jev", func(t *testing.T) {
		decision := Select("", testBinding)
		require.Equal(t, PathJEV, decision.Effective)
		require.Equal(t, PathJEV, decision.Preferred)
		require.False(t, decision.FellBack)
		require.False(t, decision.Suspect)
		require.Equal(t, testBinding.Digest(), decision.Binding)
	})

	for _, state := range integration.States() {
		t.Run(string(state), func(t *testing.T) {
			decision := Select(state, testBinding)
			require.Equal(t, PathJEV, decision.Preferred, "the preference is always recorded")
			require.Equal(t, state, decision.Reason)
			require.Equal(t, testBinding.Digest(), decision.Binding)
			require.Equal(t, PathMain, decision.Effective, "every state falls back to main: the provider never blocks a handoff")
			require.Equal(t, state != integration.StateDisabled, decision.FellBack, "disabled is a choice, not a fallback")
			require.Equal(t, state == integration.StateBindingIntegrity, decision.Suspect, "only an integrity failure is flagged suspect")
		})
	}
}

func TestEvaluateReturnsTheFirstFailingCheck(t *testing.T) {
	ok := func() integration.State { return "" }
	fail := func(state integration.State) Check { return func() integration.State { return state } }

	require.Equal(t, integration.State(""), Evaluate(ok, ok))
	require.Equal(t, integration.StateDisabled, Evaluate(ok, fail(integration.StateDisabled), fail(integration.StateTimeout)))
	require.Equal(t, integration.State(""), Evaluate())
}

func TestBreakerOpensAfterConsecutiveFailuresAndHalfOpensAfterCooldown(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	breaker := NewBreaker(2, time.Minute, func() time.Time { return now })
	timeout := integration.NewError(integration.StateTimeout, integration.Correlation{}, "late")

	require.Empty(t, breaker.Check())
	breaker.Record(timeout)
	require.Empty(t, breaker.Check(), "one failure is below the threshold")
	breaker.Record(timeout)
	require.Equal(t, integration.StateUnavailable, breaker.Check(), "open after the threshold")

	now = now.Add(2 * time.Minute)
	require.Empty(t, breaker.Check(), "half-open after the cooldown lets one probe through")
	breaker.Record(timeout)
	require.Equal(t, integration.StateUnavailable, breaker.Check(), "a failed probe reopens")

	now = now.Add(2 * time.Minute)
	breaker.Record(nil)
	require.Empty(t, breaker.Check(), "a success closes the circuit")
}

func TestBreakerIgnoresLocalAndUncountedFailures(t *testing.T) {
	breaker := NewBreaker(1, time.Minute, time.Now)
	for _, state := range []integration.State{integration.StateBindingIntegrity, integration.StateDataPolicyDenied, integration.StateUnsupportedCapability, integration.StateDisabled, integration.StateCredentialMissing} {
		breaker.Record(integration.NewError(state, integration.Correlation{}, "x"))
	}
	breaker.Record(errors.New("not an integration error"))
	require.Empty(t, breaker.Check(), "local failures say nothing about provider health")
}

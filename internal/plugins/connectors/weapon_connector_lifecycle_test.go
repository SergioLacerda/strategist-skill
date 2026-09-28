package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func readyInvoker(context.Context, InvocationEnvelope) ConnectorResult {
	return ConnectorResult{Status: domain.ReadinessReady}
}

// weaponConnector is the surface shared by the embedded and host Weapon
// connectors that these lifecycle tests exercise.
type weaponConnector interface {
	Resolve(context.Context, RuntimeLocator) ConnectorResult
	Probe(context.Context, domain.InstalledInstance, string) ConnectorResult
	Invoke(context.Context, InvocationEnvelope) ConnectorResult
	Remove(context.Context, domain.InstalledInstance) ConnectorResult
	Observe(context.Context, domain.InstalledInstance) ObservationResult
}

func weaponConnectorCases() map[string]struct {
	with, without weaponConnector
	locator       RuntimeLocator
	prefix        string
} {
	return map[string]struct {
		with, without weaponConnector
		locator       RuntimeLocator
		prefix        string
	}{
		"embedded": {
			with: EmbeddedWeaponConnector{Invoker: readyInvoker}, without: EmbeddedWeaponConnector{},
			locator: RuntimeLocator{ID: "brainstorming"}, prefix: "embedded_weapon_",
		},
		"host": {
			with: HostWeaponConnector{HostAPI: "strategist-host-skill/v1", Invoker: readyInvoker}, without: HostWeaponConnector{HostAPI: "strategist-host-skill/v1"},
			locator: RuntimeLocator{ID: "brainstorming", Path: "skills/brainstorming"}, prefix: "host_weapon_",
		},
	}
}

func TestWeaponConnectorsResolveAndProbe(t *testing.T) {
	ctx := context.Background()
	instance := domain.InstalledInstance{ID: "brainstorming"}
	for name, tc := range weaponConnectorCases() {
		t.Run(name, func(t *testing.T) {
			resolved := tc.with.Resolve(ctx, tc.locator)
			require.Equal(t, domain.ReadinessReady, resolved.Status)
			require.Equal(t, tc.prefix+"resolved", resolved.ReasonCode)
			require.Equal(t, domain.ReadinessBlocked, tc.with.Resolve(ctx, RuntimeLocator{}).Status)

			probed := tc.with.Probe(ctx, instance, "discover")
			require.Equal(t, tc.prefix+"probe_ready", probed.ReasonCode)
			require.Equal(t, "probe_input_incomplete", tc.with.Probe(ctx, domain.InstalledInstance{}, "discover").ReasonCode)
			require.Equal(t, "probe_input_incomplete", tc.with.Probe(ctx, instance, " ").ReasonCode)
			require.Equal(t, tc.prefix+"invoker_unavailable", tc.without.Probe(ctx, instance, "discover").ReasonCode)
		})
	}
}

func TestWeaponConnectorsNeverOwnLifecycleOrObservation(t *testing.T) {
	ctx := context.Background()
	for name, tc := range weaponConnectorCases() {
		t.Run(name, func(t *testing.T) {
			removed := tc.with.Remove(ctx, domain.InstalledInstance{ID: "brainstorming"})
			require.Equal(t, domain.ReadinessUnsupported, removed.Status)
			require.Equal(t, tc.prefix+"remove_not_owned", removed.ReasonCode)
			observed := tc.with.Observe(ctx, domain.InstalledInstance{ID: "brainstorming"})
			require.Equal(t, tc.prefix+"observe_not_supported", observed.ReasonCode)
		})
	}
}

func TestWeaponConnectorsRejectIncompleteInvocationContext(t *testing.T) {
	for name, tc := range weaponConnectorCases() {
		t.Run(name, func(t *testing.T) {
			result := tc.with.Invoke(context.Background(), InvocationEnvelope{Role: "ranger"})
			require.Equal(t, "invocation_context_incomplete", result.ReasonCode)
		})
	}
}

func TestEmbeddedWeaponConnectorKeepsProvidedEvidence(t *testing.T) {
	connector := EmbeddedWeaponConnector{Invoker: func(context.Context, InvocationEnvelope) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady, InvocationEvidence: "own-evidence"}
	}}
	result := connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "w"}, Role: "ranger", Slot: "discovery", MissionID: "m"})
	require.Equal(t, "own-evidence", result.InvocationEvidence)
}

func TestHostWeaponConnectorRejectsIncompatibleHostAPI(t *testing.T) {
	connector := HostWeaponConnector{HostAPI: "strategist-host-skill/v1", Invoker: readyInvoker}
	result := connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "w"}, Role: "ranger", Slot: "discovery", MissionID: "m", HostAPI: "other/v9"})
	require.Equal(t, "host_api_incompatible", result.ReasonCode)
}

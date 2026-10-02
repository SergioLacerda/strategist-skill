package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedWeaponDispatchResolvesOnlyRegisteredInvokers(t *testing.T) {
	t.Parallel()
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"brainstorming@1.0.0": func(_ context.Context, envelope InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: envelope.Instance.ID}
		},
	})
	require.NoError(t, err)

	connector, err := dispatch.Connector("brainstorming", "1.0.0")
	require.NoError(t, err)
	require.True(t, connector.Capabilities(context.Background()).CanInvoke)

	_, err = dispatch.Connector("openspec-propose", "1.0.0")
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
}

func TestEmbeddedWeaponDispatchRejectsMissingInvokerAtConstruction(t *testing.T) {
	t.Parallel()
	_, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{"brainstorming@1.0.0": nil})
	require.ErrorContains(t, err, "invoker for")
}

func TestEmbeddedWeaponDispatchProbeBlocksUnregisteredWeapon(t *testing.T) {
	t.Parallel()
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"brainstorming@1.0.0": func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	})
	require.NoError(t, err)
	result := dispatch.Probe(context.Background(), "missing", "1.0.0", "discover")
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "embedded_weapon_dispatch_unavailable", result.ReasonCode)
}

func TestEmbeddedWeaponDispatchResolvesEachVersionOfOneWeaponSeparately(t *testing.T) {
	t.Parallel()
	invoker := func(tag string) EmbeddedWeaponInvoker {
		return func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: tag}
		}
	}
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"brainstorming@1.4.0": invoker("v1"),
		"brainstorming@2.0.0": invoker("v2"),
	})
	require.NoError(t, err)

	for version, want := range map[string]string{"1.4.0": "v1", "2.0.0": "v2"} {
		connector, err := dispatch.Connector("brainstorming", version)
		require.NoError(t, err)
		require.Equal(t, want, connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "brainstorming"}, Role: "ranger", Slot: "discovery", MissionID: "m"}).ProviderID)
	}

	_, err = dispatch.Connector("brainstorming", "")
	require.ErrorContains(t, err, "brainstorming@", "an omitted version never resolves to some version")
	_, err = dispatch.Connector("brainstorming", "3.0.0")
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
}

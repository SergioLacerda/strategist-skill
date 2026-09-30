package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestResolveRoleWeaponConnector_RankedUsesEmbeddedDispatch(t *testing.T) {
	t.Parallel()
	dispatch, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{
		"brainstorming@1.0.0": func(context.Context, InvocationEnvelope) ConnectorResult {
			return ConnectorResult{Status: domain.ReadinessReady}
		},
	})
	require.NoError(t, err)

	connector, err := ResolveRoleWeaponConnector(context.Background(), domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeEmbedded,
	}, dispatch, nil)
	require.NoError(t, err)
	require.Equal(t, domain.RankedRuntimeEmbedded, connector.Capabilities(context.Background()).RuntimeKind)
}

func TestResolveRoleWeaponConnector_RankedNeverFallsBackToCustom(t *testing.T) {
	t.Parallel()
	_, err := NewEmbeddedWeaponDispatch(map[string]EmbeddedWeaponInvoker{})
	require.Error(t, err)
	_, err = ResolveRoleWeaponConnector(context.Background(), domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeEmbedded,
	}, EmbeddedWeaponDispatch{}, HostWeaponConnector{Invoker: func(context.Context, InvocationEnvelope) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady}
	}})
	require.ErrorContains(t, err, "embedded_weapon_dispatch_unavailable")
}

func TestResolveRoleWeaponConnector_CustomRequiresExplicitInvoker(t *testing.T) {
	t.Parallel()
	_, err := ResolveRoleWeaponConnector(context.Background(), domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "third-party", Mode: domain.SlotBindingModeCustom,
	}, EmbeddedWeaponDispatch{}, nil)
	require.ErrorContains(t, err, "custom_runtime_connector_unavailable")
}

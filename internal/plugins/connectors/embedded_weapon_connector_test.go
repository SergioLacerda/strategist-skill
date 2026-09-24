package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedWeaponConnectorInvokesInProcessWithoutHostAPI(t *testing.T) {
	var got InvocationEnvelope
	connector := EmbeddedWeaponConnector{
		ConnectorID: "embedded-brainstorming",
		Invoker: func(_ context.Context, envelope InvocationEnvelope) ConnectorResult {
			got = envelope
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: envelope.Instance.ID}
		},
	}

	require.True(t, connector.Capabilities(context.Background()).CanInvoke)
	result := connector.Invoke(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"},
		Role:     "ranger", Slot: "discovery", Entrypoint: "discover", MissionID: "mission-1",
	})
	require.Equal(t, domain.ReadinessReady, result.Status)
	require.Equal(t, "brainstorming", got.Instance.ID)
	require.Equal(t, "embedded-weapon:brainstorming", result.InvocationEvidence)
}

func TestEmbeddedWeaponConnectorRejectsHostAPI(t *testing.T) {
	connector := EmbeddedWeaponConnector{Invoker: func(context.Context, InvocationEnvelope) ConnectorResult {
		return ConnectorResult{Status: domain.ReadinessReady}
	}}
	result := connector.Invoke(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"},
		Role:     "ranger", Slot: "discovery", MissionID: "mission-1", HostAPI: "strategist-host-skill/v1",
	})
	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "embedded_host_api_forbidden", result.ReasonCode)
}

func TestEmbeddedWeaponConnectorFailsClosedWithoutInvoker(t *testing.T) {
	connector := EmbeddedWeaponConnector{ConnectorID: "embedded-brainstorming"}
	result := connector.Invoke(context.Background(), InvocationEnvelope{
		Instance: domain.InstalledInstance{ID: "brainstorming"}, Role: "ranger", Slot: "discovery", MissionID: "mission-1",
	})
	require.Equal(t, domain.ReadinessUnsupported, result.Status)
	require.Equal(t, "embedded_weapon_invoker_unavailable", result.ReasonCode)
}

package connectors

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostWeaponConnectorInvokesInjectedHostWithCompleteEnvelope(t *testing.T) {
	var got InvocationEnvelope
	connector := HostWeaponConnector{
		ConnectorID: "fake-host", ConnectorAPIVersion: "connector/v1", HostAPI: "strategist-host-skill/v1",
		Invoker: func(_ context.Context, envelope InvocationEnvelope) ConnectorResult {
			got = envelope
			return ConnectorResult{Status: domain.ReadinessReady, ProviderID: envelope.Instance.ID, InvocationEvidence: "evidence-1"}
		},
	}

	assert.True(t, connector.Supports("strategist-host-skill/v1"))
	assert.True(t, connector.Capabilities(context.Background()).CanInvoke)
	result := connector.Invoke(context.Background(), InvocationEnvelope{
		SchemaVersion: "weapon-invocation/v1", Instance: domain.InstalledInstance{ID: "brainstorming"},
		WeaponID: "discovery-suite", ComponentID: "brainstorming", ParentInvocationID: "mission-1/discovery-suite",
		HostAPI: "strategist-host-skill/v1",
		Role:    "ranger", Slot: "discovery", Entrypoint: "discover", MissionID: "mission-1", WriteScope: ".analysis/pending/mission-1.md",
	})
	require.Equal(t, domain.ReadinessReady, result.Status)
	assert.Equal(t, "discovery-suite", got.WeaponID)
	assert.Equal(t, "brainstorming", got.ComponentID)
	assert.Equal(t, "mission-1/discovery-suite", got.ParentInvocationID)
	assert.Equal(t, ".analysis/pending/mission-1.md", got.WriteScope)
}

func TestHostWeaponConnectorFailsClosedWithoutInvoker(t *testing.T) {
	connector := HostWeaponConnector{ConnectorID: "fake-host", HostAPI: "strategist-host-skill/v1"}
	assert.False(t, connector.Capabilities(context.Background()).CanInvoke)
	result := connector.Invoke(context.Background(), InvocationEnvelope{Instance: domain.InstalledInstance{ID: "skill"}, Role: "ranger", Slot: "discovery", MissionID: "mission"})
	assert.Equal(t, domain.ReadinessUnsupported, result.Status)
	assert.Equal(t, "host_weapon_invoker_unavailable", result.ReasonCode)
}

func TestHostWeaponConnectorRejectsIncompleteContext(t *testing.T) {
	connector := HostWeaponConnector{Invoker: func(context.Context, InvocationEnvelope) ConnectorResult { return ConnectorResult{} }}
	result := connector.Invoke(context.Background(), InvocationEnvelope{MissionID: "mission"})
	assert.Equal(t, domain.ReadinessBlocked, result.Status)
	assert.Equal(t, "invocation_context_incomplete", result.ReasonCode)
}

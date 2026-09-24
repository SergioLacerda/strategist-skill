package provider

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type discoveryConnector struct {
	connectors.UnsupportedConnector
	caps      connectors.RuntimeCapabilities
	result    connectors.ConnectorResult
	invoked   bool
	lastInput connectors.InvocationEnvelope
}

func (c *discoveryConnector) Capabilities(context.Context) connectors.RuntimeCapabilities {
	return c.caps
}

func (c *discoveryConnector) Invoke(_ context.Context, envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
	c.invoked = true
	c.lastInput = envelope
	return c.result
}

func TestInvokeDiscoveryViaConnectorBridgesHostResultToRanger(t *testing.T) {
	t.Parallel()

	connector := &discoveryConnector{
		caps: connectors.RuntimeCapabilities{ConnectorID: "host-loader", CanInvoke: true},
		result: connectors.ConnectorResult{
			Status:             domain.ReadinessReady,
			ProviderID:         "brainstorming",
			InvocationEvidence: "host-run-42",
			Artifact:           []byte("# Findings\n\nUntrusted result."),
		},
	}
	got, err := InvokeDiscoveryViaConnector(context.Background(), validDiscoveryRequest(), domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-1")
	require.NoError(t, err)
	assert.True(t, connector.invoked)
	assert.Equal(t, "ranger", connector.lastInput.Role)
	assert.Equal(t, "discovery", connector.lastInput.Slot)
	assert.Equal(t, "discover", connector.lastInput.Entrypoint)
	assert.Equal(t, "mission-1", connector.lastInput.MissionID)
	assert.Equal(t, validDiscoveryRequest().ArtifactPath, connector.lastInput.ArtifactPath)
	assert.Equal(t, "host-run-42", got.InvocationEvidence)
	assert.Contains(t, string(got.Content), "mission_status: ranger_pending")
}

func TestInvokeDiscoveryViaConnectorFailsClosedWhenConnectorCannotInvoke(t *testing.T) {
	t.Parallel()

	connector := &discoveryConnector{caps: connectors.RuntimeCapabilities{ConnectorID: "static-loader"}}
	_, err := InvokeDiscoveryViaConnector(context.Background(), validDiscoveryRequest(), domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "cannot invoke")
	assert.False(t, connector.invoked)
}

func TestInvokeDiscoveryViaConnectorRejectsIncompleteHostResult(t *testing.T) {
	t.Parallel()

	connector := &discoveryConnector{
		caps:   connectors.RuntimeCapabilities{ConnectorID: "host-loader", CanInvoke: true},
		result: connectors.ConnectorResult{Status: domain.ReadinessReady, ProviderID: "brainstorming"},
	}
	_, err := InvokeDiscoveryViaConnector(context.Background(), validDiscoveryRequest(), domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "invocation evidence is required")
}

func TestInvokeDiscoveryViaConnectorRejectsInstanceMismatch(t *testing.T) {
	t.Parallel()

	connector := &discoveryConnector{
		caps: connectors.RuntimeCapabilities{ConnectorID: "host-loader", CanInvoke: true},
		result: connectors.ConnectorResult{
			Status:             domain.ReadinessReady,
			ProviderID:         "brainstorming",
			InvocationEvidence: "host-run-42",
			Artifact:           []byte("# Findings"),
		},
	}
	_, err := InvokeDiscoveryViaConnector(context.Background(), validDiscoveryRequest(), domain.InstalledInstance{ID: "other-provider"}, connector, nil, "run-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_invocation_failed")
	assert.Contains(t, err.Error(), "does not match selected Weapon")
	assert.False(t, connector.invoked)
}

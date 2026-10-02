package provider

import (
	"context"
	"testing"
	"time"

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
			InvocationReceipt:  connectors.InvocationReceipt{SchemaVersion: connectors.InvocationReceiptSchemaVersion, MissionID: "mission-1", Role: "ranger", ProviderID: "brainstorming", ResolvedLocation: "skills/brainstorming/SKILL.md", ResolvedDigest: "sha256:test", Nonce: "connector-1", IssuedAt: time.Now(), CapabilityIsolation: connectors.CapabilityIsolationUnverified},
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

func TestInvokeDiscoveryViaConnectorAcceptsEmbeddedEvidenceWithoutHostReceipt(t *testing.T) {
	t.Parallel()

	request := validDiscoveryRequest()
	request.BindingDigest = "sha256:binding"
	request.SourceDigest = "sha256:source"
	connector := &discoveryConnector{
		caps: connectors.RuntimeCapabilities{ConnectorID: "embedded-runtime", RuntimeKind: domain.RankedRuntimeEmbedded, CanInvoke: true},
		result: connectors.ConnectorResult{
			Status:             domain.ReadinessReady,
			ProviderID:         "brainstorming",
			InvocationEvidence: "embedded-weapon:brainstorming",
			Artifact:           []byte("# Findings\n\nUntrusted result."),
			EmbeddedInvocationReceipt: connectors.EmbeddedInvocationReceipt{
				SchemaVersion: connectors.EmbeddedInvocationReceiptSchemaVersion, MissionID: "mission-1", Role: "ranger", Slot: "discovery", WeaponID: "brainstorming",
				Entrypoint: "discover", BindingDigest: "sha256:binding", SourceDigest: "sha256:source", IssuedAt: time.Now(),
			},
		},
	}

	got, err := InvokeDiscoveryViaConnector(context.Background(), request, domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-embedded")
	require.NoError(t, err)
	assert.Equal(t, "embedded-weapon:brainstorming", got.InvocationEvidence)
}

func TestInvokeDiscoveryViaRoleWeaponBindingUsesEmbeddedDispatch(t *testing.T) {
	t.Parallel()

	dispatch, err := connectors.NewEmbeddedWeaponDispatch(map[string]connectors.EmbeddedWeaponInvoker{
		"brainstorming@1.0.0": func(_ context.Context, envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
			return connectors.ConnectorResult{
				Status: domain.ReadinessReady, ProviderID: envelope.Instance.ID, InvocationEvidence: "embedded-binding", Artifact: []byte("# Findings"),
				EmbeddedInvocationReceipt: connectors.EmbeddedInvocationReceipt{
					SchemaVersion: connectors.EmbeddedInvocationReceiptSchemaVersion, MissionID: envelope.MissionID, Role: envelope.Role, Slot: envelope.Slot, WeaponID: envelope.WeaponID,
					Entrypoint: envelope.Entrypoint, BindingDigest: envelope.BindingDigest, SourceDigest: envelope.SourceDigest, IssuedAt: time.Now(),
				},
			}
		},
	})
	require.NoError(t, err)
	request := validDiscoveryRequest()
	got, err := InvokeDiscoveryViaRoleWeaponBinding(context.Background(), request, domain.InstalledInstance{ID: "brainstorming"}, domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeEmbedded, BindingDigest: "sha256:binding", SourceDigest: "sha256:source", Entrypoint: "discover",
	}, dispatch, nil, nil, "binding-run")
	require.NoError(t, err)
	assert.Equal(t, "embedded-binding", got.InvocationEvidence)
}

func TestInvokeDiscoveryViaRoleWeaponBindingDoesNotFallbackWhenEmbeddedDispatchIsMissing(t *testing.T) {
	t.Parallel()

	dispatch, err := connectors.NewEmbeddedWeaponDispatch(map[string]connectors.EmbeddedWeaponInvoker{
		"other": func(context.Context, connectors.InvocationEnvelope) connectors.ConnectorResult {
			return connectors.ConnectorResult{}
		},
	})
	require.NoError(t, err)
	_, err = InvokeDiscoveryViaRoleWeaponBinding(context.Background(), validDiscoveryRequest(), domain.InstalledInstance{ID: "brainstorming"}, domain.RoleWeaponBinding{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeEmbedded,
	}, dispatch, &discoveryConnector{caps: connectors.RuntimeCapabilities{CanInvoke: true}}, nil, "binding-run")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedded_weapon_dispatch_unavailable")
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

func TestInvokeDiscoveryViaConnectorAcceptsVerifiedIsolationOnlyFromEnforcingHost(t *testing.T) {
	t.Parallel()

	request := validDiscoveryRequest()
	connector := &discoveryConnector{
		caps: connectors.RuntimeCapabilities{ConnectorID: "scoped-host", CanInvoke: true, CanEnforcePermissions: true},
		result: connectors.ConnectorResult{
			Status: domain.ReadinessReady, ProviderID: "brainstorming", InvocationEvidence: "host-run-verified", Artifact: []byte("# Findings"),
			InvocationReceipt: connectors.InvocationReceipt{SchemaVersion: connectors.InvocationReceiptSchemaVersion, MissionID: request.MissionID, Role: request.Role, ProviderID: request.ProviderID, ResolvedLocation: "skills/brainstorming/SKILL.md", ResolvedDigest: "sha256:test", Nonce: "verified-isolation", IssuedAt: time.Now(), CapabilityIsolation: connectors.CapabilityIsolationVerified},
		},
	}
	_, err := InvokeDiscoveryViaConnector(context.Background(), request, domain.InstalledInstance{ID: "brainstorming"}, connector, nil, "run-1")
	require.NoError(t, err)
}

package mission

import (
	"context"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/stretchr/testify/require"
)

type missionDiscoveryConnector struct {
	connectors.UnsupportedConnector
	called bool
}

func (c *missionDiscoveryConnector) Capabilities(context.Context) connectors.RuntimeCapabilities {
	return connectors.RuntimeCapabilities{ConnectorID: "mission-host", CanInvoke: true}
}

func (c *missionDiscoveryConnector) Invoke(_ context.Context, envelope connectors.InvocationEnvelope) connectors.ConnectorResult {
	c.called = true
	if envelope.Role != "ranger" || envelope.Slot != string(domain.SlotDiscovery) || envelope.Entrypoint != "discover" {
		return connectors.ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "mission_context_invalid"}
	}
	return connectors.ConnectorResult{
		Status: domain.ReadinessReady, ProviderID: envelope.Instance.ID,
		InvocationEvidence: "mission-host-run-1", Artifact: []byte("# Untrusted findings\n\nbody"),
		InvocationReceipt: connectors.InvocationReceipt{SchemaVersion: connectors.InvocationReceiptSchemaVersion, MissionID: envelope.MissionID, Role: envelope.Role, ProviderID: envelope.Instance.ID, ResolvedLocation: "skills/brainstorming/SKILL.md", ResolvedDigest: "sha256:test", Nonce: "mission-1", IssuedAt: time.Now(), CapabilityIsolation: connectors.CapabilityIsolationUnverified},
	}
}

func TestInvokeRangerDiscoveryBuildsTrustedRoleBoundaryAndNormalizesResult(t *testing.T) {
	connector := &missionDiscoveryConnector{}
	got, err := InvokeRangerDiscoveryWithReceiptStore(context.Background(), "mission-1", "brainstorming", ".analysis/pending/mission-1.md", domain.InstalledInstance{ID: "brainstorming"}, connector, provider.NewMemoryReceiptNonceStore(), nil, "run-1")
	require.NoError(t, err)
	require.True(t, connector.called)
	require.Equal(t, "brainstorming", got.ProviderID)
	require.Equal(t, "mission-host-run-1", got.InvocationEvidence)
	require.Contains(t, string(got.Content), "mission_status: ranger_pending")
}

func TestInvokeRangerDiscoveryFailsClosedWithoutConnector(t *testing.T) {
	_, err := InvokeRangerDiscovery(context.Background(), "mission-1", "brainstorming", ".analysis/pending/mission-1.md", domain.InstalledInstance{ID: "brainstorming"}, nil, nil, "run-1")
	require.ErrorContains(t, err, "role_invocation_failed")
}

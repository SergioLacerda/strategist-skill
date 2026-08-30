package connectors_test

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedConnectorReturnsTypedUnsupportedResults(t *testing.T) {
	t.Parallel()

	connector := connectors.UnsupportedConnector{IDValue: "codex-unmanaged", ConnectorAPIVersion: "strategist-connector-api/1"}
	caps := connector.Capabilities(context.Background())
	assert.Equal(t, "codex-unmanaged", caps.ConnectorID)
	assert.False(t, caps.CanInvoke)
	assert.False(t, caps.CanEnforcePermissions)

	instance := domain.InstalledInstance{ID: "inst-1"}
	resolved := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "brainstorming"})
	assert.Equal(t, domain.ReadinessUnsupported, resolved.Status)
	assert.Equal(t, "connector_unsupported", resolved.ReasonCode)

	probe := connector.Probe(context.Background(), instance, "discover")
	assert.Equal(t, domain.ReadinessUnsupported, probe.Status)
	assert.Equal(t, "probe_unsupported", probe.ReasonCode)

	invoked := connector.Invoke(context.Background(), connectors.InvocationEnvelope{Instance: instance, Entrypoint: "discover"})
	assert.Equal(t, domain.ReadinessUnsupported, invoked.Status)
	assert.Equal(t, "invoke_unsupported", invoked.ReasonCode)

	removed := connector.Remove(context.Background(), instance)
	assert.Equal(t, domain.ReadinessUnsupported, removed.Status)
	assert.Equal(t, "remove_unsupported", removed.ReasonCode)

	observed := connector.Observe(context.Background(), instance)
	assert.Equal(t, domain.ReadinessUnsupported, observed.Status)
	assert.Equal(t, "observe_unsupported", observed.ReasonCode)
	assert.Equal(t, "codex-unmanaged", observed.Enforcement.ConnectorID)
	assert.Contains(t, observed.Enforcement.Limitations, "enforcement_not_supported")
}

func TestReadinessVectorDoesNotTreatUnsupportedAsReady(t *testing.T) {
	t.Parallel()

	vector := domain.PluginReadinessVector{
		Descriptor:          domain.ReadinessCheck{Status: domain.ReadinessReady},
		Source:              domain.ReadinessCheck{Status: domain.ReadinessReady},
		Trust:               domain.ReadinessCheck{Status: domain.ReadinessReady},
		Dependencies:        domain.ReadinessCheck{Status: domain.ReadinessReady},
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessReady},
		Connector:           domain.ReadinessCheck{Status: domain.ReadinessUnsupported, ReasonCode: "connector_unsupported"},
		Entrypoint:          domain.ReadinessCheck{Status: domain.ReadinessUnknown},
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessReady},
		EnforcementCoverage: domain.ReadinessCheck{Status: domain.ReadinessUnsupported, ReasonCode: "enforcement_unsupported"},
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady},
	}

	assert.False(t, vector.Ready())
	assert.Contains(t, vector.ReasonCodes(), "connector_unsupported")
	assert.Contains(t, vector.ReasonCodes(), "enforcement_unsupported")
}

func TestNativeRuntimeConnectorRejectsIncompleteLocatorAndProbeInput(t *testing.T) {
	t.Parallel()

	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native"}

	blockedResolve := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "sniper"}) // no Path
	assert.Equal(t, domain.ReadinessBlocked, blockedResolve.Status)
	assert.Equal(t, "locator_incomplete", blockedResolve.ReasonCode)

	blockedProbe := connector.Probe(context.Background(), domain.InstalledInstance{}, "materialize_docs") // no instance ID
	assert.Equal(t, domain.ReadinessBlocked, blockedProbe.Status)
	assert.Equal(t, "probe_input_incomplete", blockedProbe.ReasonCode)
}

func TestNativeRuntimeConnectorRemoveIsNotOwned(t *testing.T) {
	t.Parallel()

	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native"}
	removed := connector.Remove(context.Background(), domain.InstalledInstance{ID: "sniper"})
	assert.Equal(t, domain.ReadinessUnsupported, removed.Status)
	assert.Equal(t, "remove_not_owned_by_static_connector", removed.ReasonCode)
}

func TestNativeRuntimeConnectorObservesEnforcementWhenConfigured(t *testing.T) {
	t.Parallel()

	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native", EnforcementObservable: true}
	caps := connector.Capabilities(context.Background())
	assert.True(t, caps.CanObserve)
	assert.True(t, caps.CanEnforcePermissions)

	observed := connector.Observe(context.Background(), domain.InstalledInstance{ID: "sniper"})
	assert.Equal(t, domain.ReadinessReady, observed.Status)
	assert.Equal(t, "enforcement_observed", observed.ReasonCode)
	assert.Contains(t, observed.Enforcement.Enforceable, domain.PluginPermissionReadWorkspace)
	assert.Empty(t, observed.Enforcement.Limitations)
}

func TestNativeRuntimeConnectorReportsVisibleLocalInstanceWithoutInvokeClaim(t *testing.T) {
	t.Parallel()

	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native", ConnectorAPIVersion: "strategist-connector-api/1"}
	caps := connector.Capabilities(context.Background())
	assert.True(t, caps.CanResolve)
	assert.True(t, caps.CanProbe)
	assert.False(t, caps.CanInvoke)
	assert.False(t, caps.CanEnforcePermissions)

	resolved := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "sniper", Path: "roles/sniper.yaml"})
	require.Equal(t, domain.ReadinessReady, resolved.Status)
	assert.Equal(t, "resolved_local_locator", resolved.ReasonCode)

	probe := connector.Probe(context.Background(), domain.InstalledInstance{ID: "sniper"}, "materialize_docs")
	assert.Equal(t, domain.ReadinessReady, probe.Status)
	assert.Equal(t, "static_probe_ready", probe.ReasonCode)

	invoked := connector.Invoke(context.Background(), connectors.InvocationEnvelope{Instance: domain.InstalledInstance{ID: "sniper"}, Entrypoint: "materialize_docs"})
	assert.Equal(t, domain.ReadinessUnsupported, invoked.Status)
	assert.Equal(t, "invoke_not_claimed_by_static_connector", invoked.ReasonCode)

	observed := connector.Observe(context.Background(), domain.InstalledInstance{ID: "sniper"})
	assert.Equal(t, domain.ReadinessUnsupported, observed.Status)
	assert.Contains(t, observed.Enforcement.Limitations, "enforcement_not_supported")
}

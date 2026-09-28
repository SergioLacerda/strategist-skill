package connectors

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
)

// NativeRuntimeConnector reports static visibility for current in-process defaults.
type NativeRuntimeConnector struct {
	ConnectorID         string
	ConnectorAPIVersion string
	// EnforcementObservable, when true, means this connector can observe
	// LOCAL write-scope policy enforcement (see Observe below) — it says
	// nothing about host-level capability isolation for a delegated role.
	// See RuntimeCapabilities.CanEnforcePermissions's doc comment for the
	// full disambiguation; the two production call sites that set this true
	// today (cmd/strategist/plugins/evaluate_write.go,
	// internal/authorization/report.go) are both write-scope checks.
	EnforcementObservable bool
}

// Capabilities reports static in-process connector abilities.
func (c NativeRuntimeConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{
		ConnectorID:           c.ConnectorID,
		ConnectorAPI:          c.ConnectorAPIVersion,
		CanResolve:            true,
		CanProbe:              true,
		CanObserve:            c.EnforcementObservable,
		CanEnforcePermissions: c.EnforcementObservable,
	}
}

// Resolve validates that a local runtime locator is complete.
func (c NativeRuntimeConnector) Resolve(_ context.Context, locator RuntimeLocator) ConnectorResult {
	if locator.ID == "" || locator.Path == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "locator_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "resolved_local_locator", Detail: locator.Path}
}

// Probe validates static probe inputs without claiming live readiness. Input
// validation is useful, but it is not evidence that an external runtime was
// reached or that its entrypoint can execute.
func (c NativeRuntimeConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if instance.ID == "" || entrypoint == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessUnknown, ReasonCode: "probe_not_verified", Detail: "static connector performed no runtime invocation"}
}

// Invoke reports that static connectors do not claim invocation authority.
func (c NativeRuntimeConnector) Invoke(context.Context, InvocationEnvelope) ConnectorResult {
	return unsupported("invoke_not_claimed_by_static_connector")
}

// Remove reports that static connectors do not own removal.
func (c NativeRuntimeConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("remove_not_owned_by_static_connector")
}

// Observe reports static enforcement evidence when configured.
func (c NativeRuntimeConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	if c.EnforcementObservable {
		return ObservationResult{
			ConnectorResult: ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "enforcement_observed"},
			Enforcement: policy.EnforcementReport{
				ConnectorID: c.ConnectorID,
				Enforceable: []domain.PluginPermission{
					domain.PluginPermissionReadWorkspace,
					domain.PluginPermissionWriteAnalysis,
				},
			},
		}
	}
	return ObservationResult{
		ConnectorResult: unsupported("enforcement_unsupported"),
		Enforcement:     policy.EnforcementReport{ConnectorID: c.ConnectorID, Limitations: []string{"enforcement_not_supported"}},
	}
}

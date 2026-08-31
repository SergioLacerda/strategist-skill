// Package connectors defines runtime plugin connector contracts.
package connectors

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
)

// RuntimeConnector is the host boundary for runtime-specific plugin operations.
type RuntimeConnector interface {
	Capabilities(context.Context) RuntimeCapabilities
	Resolve(context.Context, RuntimeLocator) ConnectorResult
	Probe(context.Context, domain.InstalledInstance, string) ConnectorResult
	Invoke(context.Context, InvocationEnvelope) ConnectorResult
	Remove(context.Context, domain.InstalledInstance) ConnectorResult
	Observe(context.Context, domain.InstalledInstance) ObservationResult
}

// RuntimeCapabilities declares what a connector can truthfully perform.
type RuntimeCapabilities struct {
	ConnectorID           string
	ConnectorAPI          string
	CanResolve            bool
	CanProbe              bool
	CanInvoke             bool
	CanRemove             bool
	CanObserve            bool
	CanEnforcePermissions bool
}

// RuntimeLocator identifies a local or installed runtime resource.
type RuntimeLocator struct {
	ID   string
	Path string
}

// InvocationEnvelope is the versioned runtime invocation input.
type InvocationEnvelope struct {
	SchemaVersion string
	Instance      domain.InstalledInstance
	Entrypoint    string
	MissionID     string
	GateAllowed   bool
}

// ConnectorResult is a typed connector response for every operation.
type ConnectorResult struct {
	Status     domain.ReadinessStatus
	ReasonCode string
	Detail     string
}

// ObservationResult includes enforcement evidence without substituting for it.
type ObservationResult struct {
	ConnectorResult
	Enforcement policy.EnforcementReport
}

// UnsupportedConnector is the safe default for runtimes without a live SPI.
type UnsupportedConnector struct {
	IDValue             string
	ConnectorAPIVersion string
}

// Capabilities reports no active operations for unsupported runtimes.
func (c UnsupportedConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{ConnectorID: c.IDValue, ConnectorAPI: c.ConnectorAPIVersion}
}

// Resolve returns an unsupported connector result.
func (c UnsupportedConnector) Resolve(context.Context, RuntimeLocator) ConnectorResult {
	return unsupported("connector_unsupported")
}

// Probe returns an unsupported probe result.
func (c UnsupportedConnector) Probe(context.Context, domain.InstalledInstance, string) ConnectorResult {
	return unsupported("probe_unsupported")
}

// Invoke returns an unsupported invocation result.
func (c UnsupportedConnector) Invoke(context.Context, InvocationEnvelope) ConnectorResult {
	return unsupported("invoke_unsupported")
}

// Remove returns an unsupported removal result.
func (c UnsupportedConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("remove_unsupported")
}

// Observe reports that enforcement observation is unsupported.
func (c UnsupportedConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	return ObservationResult{
		ConnectorResult: unsupported("observe_unsupported"),
		Enforcement:     policy.EnforcementReport{ConnectorID: c.IDValue, Limitations: []string{"enforcement_not_supported"}},
	}
}

// NativeRuntimeConnector reports static visibility for current in-process defaults.
type NativeRuntimeConnector struct {
	ConnectorID           string
	ConnectorAPIVersion   string
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

// Probe validates static probe inputs without invoking external code.
func (c NativeRuntimeConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if instance.ID == "" || entrypoint == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "static_probe_ready"}
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

func unsupported(reason string) ConnectorResult {
	return ConnectorResult{Status: domain.ReadinessUnsupported, ReasonCode: reason}
}

// NativeRoleConnector lives in native_role_connector.go, split out to keep
// this file under the repo's file-size budget.

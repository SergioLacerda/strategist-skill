package connectors

import (
	"context"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EmbeddedWeaponInvoker executes a Strategist-owned skill in the current
// process/context. It is deliberately distinct from HostWeaponInvoker: an
// embedded Ranked weapon never crosses a host loader boundary and never
// accepts a host API declaration.
type EmbeddedWeaponInvoker func(context.Context, InvocationEnvelope) ConnectorResult

// EmbeddedWeaponConnector adapts a compiled or in-process Strategist skill to
// RuntimeConnector. The callback is an in-process capability supplied by the
// embedded runtime, not a provider-root or host-loader lookup.
type EmbeddedWeaponConnector struct {
	ConnectorID         string
	ConnectorAPIVersion string
	Invoker             EmbeddedWeaponInvoker
}

// Capabilities reports the in-process capabilities; it can invoke only when an invoker is supplied.
func (c EmbeddedWeaponConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{
		ConnectorID: c.ConnectorID, ConnectorAPI: c.ConnectorAPIVersion,
		CanResolve: true, CanProbe: true, CanInvoke: c.Invoker != nil,
	}
}

// Resolve resolves an embedded Weapon by its locator ID without a provider-root lookup.
func (c EmbeddedWeaponConnector) Resolve(_ context.Context, locator RuntimeLocator) ConnectorResult {
	if strings.TrimSpace(locator.ID) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_locator_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "embedded_weapon_resolved", ProviderID: locator.ID}
}

// Probe probes that the instance and entrypoint are declared and an invoker is available.
func (c EmbeddedWeaponConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(entrypoint) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	if c.Invoker == nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_weapon_invoker_unavailable"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "embedded_weapon_probe_ready", ProviderID: instance.ID}
}

// Invoke runs the embedded Weapon in-process and records its invocation evidence.
func (c EmbeddedWeaponConnector) Invoke(ctx context.Context, envelope InvocationEnvelope) ConnectorResult {
	if c.Invoker == nil {
		return unsupported("embedded_weapon_invoker_unavailable")
	}
	if strings.TrimSpace(envelope.Instance.ID) == "" || strings.TrimSpace(envelope.Role) == "" || strings.TrimSpace(envelope.Slot) == "" || strings.TrimSpace(envelope.MissionID) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "invocation_context_incomplete"}
	}
	if strings.TrimSpace(envelope.HostAPI) != "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_host_api_forbidden"}
	}
	result := c.Invoker(ctx, envelope)
	if result.Status == domain.ReadinessReady && strings.TrimSpace(result.InvocationEvidence) == "" {
		result.InvocationEvidence = "embedded-weapon:" + envelope.Instance.ID
	}
	return result
}

// Remove never removes anything: the Strategist runtime owns embedded Weapons.
func (c EmbeddedWeaponConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("embedded_weapon_remove_not_owned")
}

// Observe reports no enforcement observation for an in-process Weapon.
func (c EmbeddedWeaponConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	return ObservationResult{ConnectorResult: unsupported("embedded_weapon_observe_not_supported")}
}

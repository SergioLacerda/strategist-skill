package connectors

import (
	"context"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// HostWeaponInvoker is the only operation a host-loaded skill needs to expose
// to Strategist. The host owns loading and execution; Strategist receives an
// opaque result plus evidence and never infers execution from metadata.
type HostWeaponInvoker func(context.Context, InvocationEnvelope) ConnectorResult

// HostWeaponConnector adapts a host skill loader to RuntimeConnector. It is
// intentionally injectable so mission tests use a fake host and never start a
// local or remote model.
type HostWeaponConnector struct {
	ConnectorID         string
	ConnectorAPIVersion string
	HostAPI             string
	Invoker             HostWeaponInvoker
}

// Capabilities reports what the connector can do; invocation requires an Invoker.
func (c HostWeaponConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{
		ConnectorID: c.ConnectorID, ConnectorAPI: c.ConnectorAPIVersion,
		CanResolve: true, CanProbe: true, CanInvoke: c.Invoker != nil,
	}
}

// Supports reports whether the connector implements the declared host API.
func (c HostWeaponConnector) Supports(hostAPI string) bool {
	return strings.TrimSpace(hostAPI) != "" && strings.TrimSpace(c.HostAPI) == strings.TrimSpace(hostAPI)
}

// Resolve accepts a locator that carries both an ID and a path.
func (c HostWeaponConnector) Resolve(_ context.Context, locator RuntimeLocator) ConnectorResult {
	if strings.TrimSpace(locator.ID) == "" || strings.TrimSpace(locator.Path) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "locator_incomplete"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "host_weapon_resolved", ProviderID: locator.ID}
}

// Probe reports readiness when the instance, entrypoint, and Invoker are present.
func (c HostWeaponConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(entrypoint) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	if c.Invoker == nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "host_weapon_invoker_unavailable"}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "host_weapon_probe_ready", ProviderID: instance.ID}
}

// Invoke validates the envelope context and host API, then delegates to the Invoker.
func (c HostWeaponConnector) Invoke(ctx context.Context, envelope InvocationEnvelope) ConnectorResult {
	if c.Invoker == nil {
		return unsupported("host_weapon_invoker_unavailable")
	}
	if strings.TrimSpace(envelope.Instance.ID) == "" || strings.TrimSpace(envelope.Role) == "" || strings.TrimSpace(envelope.Slot) == "" || strings.TrimSpace(envelope.MissionID) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "invocation_context_incomplete"}
	}
	if !c.Supports(envelope.HostAPI) {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "host_api_incompatible"}
	}
	return c.Invoker(ctx, envelope)
}

// Remove is unsupported: the host owns skill lifecycle.
func (c HostWeaponConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("host_weapon_remove_not_owned")
}

// Observe is unsupported for host skills.
func (c HostWeaponConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	return ObservationResult{ConnectorResult: unsupported("host_weapon_observe_not_supported")}
}

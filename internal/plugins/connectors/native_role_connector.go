package connectors

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// NativeRoleConnector extends NativeRuntimeConnector to claim real invocation
// authority for the one case that is genuinely, statically verifiable in
// this codebase: a native Strategist role (roles/*.yaml — Ranger, Archivist,
// Sniper) being "invoked" by having the parent agent embody that role
// directly. That invocation path is deterministic and requires no external
// process, subprocess, or network call — the parent agent session already
// holds every capability the role needs, and by the time Invoke is called
// the role definition has already been parsed and structurally validated
// (see internal/check/check_slots.go's resolveNativeRoleSlot, which runs
// domain.RoleConfig.Validate() and confirms the role's declared slot before
// any readiness/invocation check is ever attempted). This is exactly how
// native roles already work at runtime today — this connector simply lets
// the plugin-readiness framework say so truthfully instead of leaving
// Invoke a permanent "not claimed" stub.
//
// This is deliberately NOT extended to external skill plugins. A skill
// plugin's actual invocation happens inside the Claude Code skill loader, a
// separate process this CLI does not control, start, or observe — claiming
// CanInvoke=true for that case from a static `strategist check` pass would
// be a fabricated guarantee, not a verified one. UnsupportedConnector (used
// today for skill providers, see check_slots.go's skillProviderReadiness)
// remains the honest connector for that case.
type NativeRoleConnector struct {
	NativeRuntimeConnector
}

// Capabilities reports the same static abilities as NativeRuntimeConnector,
// plus CanInvoke — the one addition this connector variant claims.
func (c NativeRoleConnector) Capabilities(ctx context.Context) RuntimeCapabilities {
	caps := c.NativeRuntimeConnector.Capabilities(ctx)
	caps.CanInvoke = true
	return caps
}

// Invoke reports the native role as invoked once the envelope is complete
// and the caller's own gate has authorized it (InvocationEnvelope.GateAllowed
// — see internal/plugins/policy.EvaluateEnforcement / EvaluateWrite for how a
// caller should decide that value before ever constructing this envelope).
// No external code is run: "invocation" here is the parent agent embodying
// the role, which is why this can be answered statically and deterministically.
func (c NativeRoleConnector) Invoke(_ context.Context, envelope InvocationEnvelope) ConnectorResult {
	if envelope.Instance.ID == "" || envelope.Entrypoint == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "invoke_input_incomplete"}
	}
	if !envelope.GateAllowed {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "invoke_gate_not_allowed", Detail: envelope.Instance.ID}
	}
	return ConnectorResult{
		Status:     domain.ReadinessReady,
		ReasonCode: "invoke_native_role_direct_embodiment",
		Detail:     "parent agent embodies role " + envelope.Instance.ID + " directly; no external process invoked",
	}
}

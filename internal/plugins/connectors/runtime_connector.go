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
	ConnectorID  string
	ConnectorAPI string
	CanResolve   bool
	CanProbe     bool
	CanInvoke    bool
	CanRemove    bool
	CanObserve   bool
	// CanEnforcePermissions reports whether this connector can observe LOCAL
	// write-scope policy enforcement (Observe, policy.EnforcementReport) — a
	// same-process, no-host-dependency check of whether a write target is
	// inside the active base_path. It is NOT evidence of host-level LLM
	// tool-session capability isolation for a delegated role (see
	// internal/provider/discovery_connector.go, which reuses this same field
	// for that unrelated meaning when populating
	// InvocationReceipt.CapabilityIsolation). The two concerns share this
	// field only by historical accident; do not read one as proof of the
	// other. See .analysis/pending/drift_pipeline/
	// 20260927-provider-boundary-host-conformance-track-a-capability-isolation.md
	// for the investigation this comment resolves.
	CanEnforcePermissions bool
}

// RuntimeLocator identifies a local or installed runtime resource.
type RuntimeLocator struct {
	ID   string
	Path string
}

// InvocationEnvelope is the versioned runtime invocation input for
// RuntimeConnector.Invoke — a host/plugin-runtime dispatch payload, not to be
// confused with domain.RoleInvocationPlan
// (internal/domain/role_invocation_plan.go), the mission-scoped Role→Weapon
// composition (pinned weapon binding + context/schema refs) a mission
// resolves before invocation. The two names are similar on purpose only in
// that both were candidates for the same English word ("envelope"/"plan") at
// different points; they are deliberately distinct types with no shared
// fields, kept separate per
// docs/adr/0041-cli-enforcement-sequencing-and-role-invocation-plan-naming.md
// D1.
type InvocationEnvelope struct {
	SchemaVersion      string
	Instance           domain.InstalledInstance
	WeaponID           string
	ComponentID        string
	ParentInvocationID string
	HostAPI            string
	Role               string
	Slot               string
	Entrypoint         string
	MissionID          string
	ArtifactPath       string
	WriteScope         string
	GateAllowed        bool
}

// ConnectorResult is a typed connector response for every operation.
type ConnectorResult struct {
	Status             domain.ReadinessStatus
	ReasonCode         string
	Detail             string
	ProviderID         string
	Artifact           []byte
	InvocationEvidence string
	InvocationReceipt  InvocationReceipt
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

func unsupported(reason string) ConnectorResult {
	return ConnectorResult{Status: domain.ReadinessUnsupported, ReasonCode: reason}
}

// InvocationReceipt and its validation live in invocation_receipt.go;
// NativeRuntimeConnector and its methods live in native_runtime_connector.go;
// NativeRoleConnector lives in native_role_connector.go — all split out to
// keep this file under the repo's file-size budget.

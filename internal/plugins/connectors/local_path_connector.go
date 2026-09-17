package connectors

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/policy"
)

// LocalPathConnector resolves ORKA-shaped skill packages (SKILL.md mandatory,
// references/, scripts/, templates/, assets/ optional — ADR-0033) from a
// filesystem directory into domain.PluginPackage values for the embedded-skill
// ingestion pipeline (.analysis/refined/20260913-embedded-skill-directory-catalog
// Task 1). It never claims invocation authority for the resolved package:
// like UnsupportedConnector (used today for external skill providers — see
// native_role_connector.go's own comment), actual invocation of an external
// skill's prompt content happens inside a separate process this CLI does not
// control, start, or observe.
type LocalPathConnector struct {
	ConnectorID         string
	ConnectorAPIVersion string
}

// Capabilities reports static resolve/probe ability only — no invoke, no
// remove, no enforcement observation, matching the honesty principle
// UnsupportedConnector already establishes for externally-invoked skills.
func (c LocalPathConnector) Capabilities(context.Context) RuntimeCapabilities {
	return RuntimeCapabilities{
		ConnectorID:  c.ConnectorID,
		ConnectorAPI: c.ConnectorAPIVersion,
		CanResolve:   true,
		CanProbe:     true,
	}
}

// Resolve reports whether locator.Path holds a structurally valid ORKA
// package, without activating or cataloguing it. ResolveLocalPackage is the
// function that actually produces the domain.PluginPackage the ingestion
// generator catalogues — this method only answers the RuntimeConnector
// contract's own resolve question.
func (c LocalPathConnector) Resolve(_ context.Context, locator RuntimeLocator) ConnectorResult {
	if locator.ID == "" || locator.Path == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "locator_incomplete"}
	}
	if _, err := ResolveLocalPackage(locator.Path); err != nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "local_package_invalid", Detail: err.Error()}
	}
	return ConnectorResult{Status: domain.ReadinessReady, ReasonCode: "resolved_local_package", Detail: locator.Path}
}

// Probe validates static probe inputs without invoking external code. Static
// validation is not runtime evidence, so a valid input remains unverified.
func (c LocalPathConnector) Probe(_ context.Context, instance domain.InstalledInstance, entrypoint string) ConnectorResult {
	if instance.ID == "" || entrypoint == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "probe_input_incomplete"}
	}
	return ConnectorResult{
		Status:     domain.ReadinessUnknown,
		ReasonCode: "probe_not_verified",
		Detail:     "local-path connector performed no runtime invocation",
	}
}

// Invoke never claims invocation authority — the resolved package's prompt
// content runs inside the host's own skill loader, a process this connector
// does not control.
func (c LocalPathConnector) Invoke(context.Context, InvocationEnvelope) ConnectorResult {
	return unsupported("invoke_not_claimed_by_local_path_connector")
}

// Remove never claims removal authority — this connector only resolves and
// reads; it never mutates the source directory it was pointed at.
func (c LocalPathConnector) Remove(context.Context, domain.InstalledInstance) ConnectorResult {
	return unsupported("remove_not_owned_by_local_path_connector")
}

// Observe reports enforcement as unsupported — matching every other
// connector variant that cannot verify runtime enforcement.
func (c LocalPathConnector) Observe(context.Context, domain.InstalledInstance) ObservationResult {
	return ObservationResult{
		ConnectorResult: unsupported("enforcement_unsupported"),
		Enforcement:     policy.EnforcementReport{ConnectorID: c.ConnectorID, Limitations: []string{"enforcement_not_supported"}},
	}
}

// requiredSkillManifestFile, skillFrontmatter, ResolveLocalPackage,
// parseSkillFrontmatter, hasPrefixTrimmed, indexOf, and digestPackageDirectory
// live in local_path_package_resolver.go, split out to keep this file under
// the repo's file-size budget.

package install

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
)

func connectorProbe(binding domain.SlotBinding, instance domain.InstalledInstance) lifecycle.ProbeOutcome {
	switch binding.RuntimeKind {
	case domain.RankedRuntimeHost:
		return hostStructuralProbe(binding, instance)
	case domain.RankedRuntimeEmbedded, domain.RankedRuntimeOpenSpecRoot:
		return internalRuntimeStructuralProbe(binding, instance)
	}
	entrypoint := "refine"
	if binding.Slot == string(domain.SlotDiscovery) {
		entrypoint = "discover"
	}
	result := connectors.LocalPathConnector{
		ConnectorID:         instance.ConnectorID,
		ConnectorAPIVersion: "strategist-connector-api/1",
	}.Probe(context.Background(), instance, entrypoint)
	return lifecycle.ProbeOutcome{Status: result.Status, ReasonCode: result.ReasonCode, Detail: result.Detail}
}

// hostStructuralProbe validates the catalog-derived host boundary needed to
// persist a binding. It deliberately does not invoke the host skill: live
// invocation evidence is collected by the host adapter at dispatch time.
func hostStructuralProbe(binding domain.SlotBinding, instance domain.InstalledInstance) lifecycle.ProbeOutcome {
	if binding.ConnectorID != "strategist-host" || instance.ConnectorID != binding.ConnectorID {
		return lifecycle.ProbeOutcome{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "host_binding_identity_incomplete",
			Detail:     "host binding connector identity does not match the catalog runtime",
		}
	}
	if binding.InstalledInstanceID == "" || binding.Entrypoint == "" || instance.ID == "" {
		return lifecycle.ProbeOutcome{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "host_binding_metadata_incomplete",
			Detail:     "host binding requires instance and entrypoint metadata",
		}
	}
	return lifecycle.ProbeOutcome{
		Status:     domain.ReadinessReady,
		ReasonCode: "host_package_structural_ready",
		Detail:     "catalog host package is structurally ready; live invocation remains unverified",
	}
}

// internalRuntimeConnectors lists, per runtime kind, the connector identities a
// Weapon whose runtime ships inside this binary may carry (ADR-0067). Anything
// else under these kinds is an identity error, not a package to probe.
var internalRuntimeConnectors = map[string][]string{
	domain.RankedRuntimeEmbedded:     {"strategist-embedded", "strategist-native-role"},
	domain.RankedRuntimeOpenSpecRoot: {"strategist-ranked-runtime"},
}

// internalRuntimeStructuralProbe validates the identity of a Weapon whose
// runtime is embedded in this binary (a prompt bridge or the private OpenSpec
// root). Like the host probe it proves structure, not invocation: a switch
// between certified Weapons would otherwise never leave the unknown state of
// the local-path probe, which only describes external packages, and the whole
// install would roll back. Live invocation is verified by `mission invoke`.
func internalRuntimeStructuralProbe(binding domain.SlotBinding, instance domain.InstalledInstance) lifecycle.ProbeOutcome {
	if !containsString(internalRuntimeConnectors[binding.RuntimeKind], binding.ConnectorID) || instance.ConnectorID != binding.ConnectorID {
		return lifecycle.ProbeOutcome{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "internal_runtime_identity_mismatch",
			Detail:     "runtime connector identity does not match the runtime kind the catalog declares",
		}
	}
	if binding.InstalledInstanceID == "" || binding.Entrypoint == "" || instance.ID == "" {
		return lifecycle.ProbeOutcome{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "internal_runtime_metadata_incomplete",
			Detail:     "runtime binding requires instance and entrypoint metadata",
		}
	}
	return lifecycle.ProbeOutcome{
		Status:     domain.ReadinessReady,
		ReasonCode: "internal_runtime_structural_ready",
		Detail:     "certified runtime ships with this binary; live invocation remains unverified",
	}
}

package check

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/weapon"
)

// readinessFacets are the dimensions that depend on where a Weapon's manifest
// lives: the catalog entry or a custom provider adapter.
type readinessFacets struct {
	descriptor domain.ReadinessCheck
	source     domain.ReadinessCheck
	entrypoint domain.ReadinessCheck
	// trustAndGrant overrides how trust and the permission grant are evaluated; nil
	// keeps the default lookup keyed by the provider id.
	trustAndGrant func(root, provider string, lock domain.PluginLockFile) (domain.ReadinessCheck, domain.ReadinessCheck)
	// hostAPI is the host API dimension; the zero value means the source
	// declares none (host_api_not_declared).
	hostAPI     domain.ReadinessCheck
	detail      string
	conformance func(probe connectors.ConnectorResult) domain.ReadinessCheck
}

func weaponReadiness(root, slot, provider string, facets readinessFacets) domain.PluginReadinessVector {
	connector := connectors.UnsupportedConnector{IDValue: "current-runtime", ConnectorAPIVersion: "strategist-connector-api/1"}
	resolve := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: provider, Path: facets.detail})
	observe := connector.Observe(context.Background(), domain.InstalledInstance{ID: provider})
	lock := readPluginsLockFile(root)
	if bindingIsRanked(lock, slot, provider) {
		// A Ranked binding is already validated and certified at build time
		// (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
		// DEC-003) — it never calls into Custom's trust.Verify/
		// policy.EvaluateGrant runtime checks; readiness is reported from the
		// catalog's certification stamp, and its installed runtime is the
		// dependencies dimension.
		trustCheck, grantCheck, runtimeCheck := rankedCertificationReadiness(root, slot, provider)
		certified := domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_certification_verified"}
		return vectorFromFacets(facets, certified, trustCheck, grantCheck, runtimeCheck, resolve, observe)
	}
	entrypoint := "refine"
	if slot == string(domain.SlotDiscovery) {
		entrypoint = "discover"
	}
	probe := connector.Probe(context.Background(), domain.InstalledInstance{ID: provider, ConnectorID: connector.Capabilities(context.Background()).ConnectorID}, entrypoint)
	trustCheck, grantCheck := facets.trustAndGrantChecks(root, provider, lock)
	// Every non-Ranked binding must still not read ready without a runtime.
	return vectorFromFacets(facets, facets.conformance(probe), trustCheck, grantCheck, customRuntimeReadiness(root, slot, provider), resolve, observe)
}

func vectorFromFacets(facets readinessFacets, conformance, trustCheck, grantCheck, dependencies domain.ReadinessCheck, resolve connectors.ConnectorResult, observe connectors.ObservationResult) domain.PluginReadinessVector {
	return domain.PluginReadinessVector{
		Descriptor:          facets.descriptor,
		Source:              facets.source,
		Conformance:         conformance,
		Trust:               trustCheck,
		Dependencies:        dependencies,
		HostAPI:             hostAPICheck(facets.hostAPI),
		Connector:           connectorCheck(resolve),
		Entrypoint:          facets.entrypoint,
		PermissionGrant:     grantCheck,
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
}

func requestedPermissions(root, provider string) []domain.PluginPermission {
	facts, err := weapon.ResolveWeaponFacts(root, provider)
	if err != nil {
		return nil
	}
	return facts.RequestedPermissions
}

// bindingIsRanked reports whether slot's persisted plugins.lock binding for
// provider has EffectiveMode() == SlotBindingModeRanked. A missing or
// mismatched binding is not Ranked — the same fail-closed default every
// other readiness dimension already uses for absent state.
func bindingIsRanked(lock domain.PluginLockFile, slot, provider string) bool {
	for _, b := range lock.Bindings {
		if b.Slot == slot && b.InstalledInstanceID == provider {
			return b.EffectiveMode() == domain.SlotBindingModeRanked
		}
	}
	return false
}

// Ranked-certification-specific readiness (rankedCertificationReadiness,
// evaluateRankedConformance, liveHostAPIDigest) lives in
// check_ranked_readiness.go, split out to keep this file under the repo's
// file-size budget.

// Blocked-readiness diagnostic aggregation (readinessDimension,
// readinessDimensions, blockedReadinessErrors, blockedReadinessErrorsForSlots)
// lives in check_readiness_errors.go, split out to keep this file under the
// repo's file-size budget.

func nativeRoleReadiness(provider, path string) domain.PluginReadinessVector {
	connector := connectors.NativeRuntimeConnector{ConnectorID: "strategist-native", ConnectorAPIVersion: "strategist-connector-api/1"}
	instance := domain.InstalledInstance{ID: provider, State: "active"}
	resolve := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: provider, Path: path})
	probe := connector.Probe(context.Background(), instance, "native_role")
	observe := connector.Observe(context.Background(), instance)
	return domain.PluginReadinessVector{
		Descriptor:          domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "native_role_valid", Detail: path},
		Source:              domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_role_present", Detail: path},
		Trust:               domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "native_baseline_trusted"},
		Dependencies:        domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "native_role_no_plugin_dependencies"},
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "native_host_api"},
		Connector:           connectorCheck(resolve),
		Entrypoint:          connectorCheck(probe),
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "native_role_no_external_grant"},
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
}

func connectorCheck(result connectors.ConnectorResult) domain.ReadinessCheck {
	return domain.ReadinessCheck{Status: result.Status, ReasonCode: result.ReasonCode, Detail: result.Detail}
}

func connectorObservationCheck(result connectors.ObservationResult) domain.ReadinessCheck {
	return domain.ReadinessCheck{Status: result.Status, ReasonCode: result.ReasonCode, Detail: result.Detail}
}

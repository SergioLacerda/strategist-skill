package check

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

// readinessFacets are the dimensions that depend on where a Weapon's manifest
// lives: the generated compat view file, or the catalog entry.
type readinessFacets struct {
	descriptor  domain.ReadinessCheck
	source      domain.ReadinessCheck
	entrypoint  domain.ReadinessCheck
	detail      string
	conformance func(probe connectors.ConnectorResult) domain.ReadinessCheck
}

// skillProviderReadiness is the readiness of a provider known only through its
// generated compat view (the transitional branch).
func skillProviderReadiness(root, slot, provider, path string) domain.PluginReadinessVector {
	return weaponReadiness(root, slot, provider, readinessFacets{
		descriptor: domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "legacy_descriptor_valid", Detail: path},
		source:     domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_manifest_present", Detail: path},
		entrypoint: probeSkillEntrypoint(provider, path),
		detail:     path,
		conformance: func(probe connectors.ConnectorResult) domain.ReadinessCheck {
			return customConformanceReadiness(root, slot, provider, path, probe)
		},
	})
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
	digest := lock.NodeDigest(provider, "adapter_contract")
	trustCheck := skillProviderTrustReadiness(root, provider, digest)
	grantCheck := skillProviderPermissionGrantReadinessFor(root, digest, requestedPermissions(root, provider))
	// Every non-Ranked binding must still not read ready without a runtime.
	return vectorFromFacets(facets, facets.conformance(probe), trustCheck, grantCheck, customRuntimeReadiness(root, slot, provider), resolve, observe)
}

func skillProviderVector(path, provider string, conformance, trustCheck, grantCheck, dependencies domain.ReadinessCheck, resolve connectors.ConnectorResult, observe connectors.ObservationResult) domain.PluginReadinessVector {
	facets := readinessFacets{
		descriptor: domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "legacy_descriptor_valid", Detail: path},
		source:     domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_manifest_present", Detail: path},
		entrypoint: probeSkillEntrypoint(provider, path),
	}
	return vectorFromFacets(facets, conformance, trustCheck, grantCheck, dependencies, resolve, observe)
}

func vectorFromFacets(facets readinessFacets, conformance, trustCheck, grantCheck, dependencies domain.ReadinessCheck, resolve connectors.ConnectorResult, observe connectors.ObservationResult) domain.PluginReadinessVector {
	return domain.PluginReadinessVector{
		Descriptor:          facets.descriptor,
		Source:              facets.source,
		Conformance:         conformance,
		Trust:               trustCheck,
		Dependencies:        dependencies,
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "host_api_not_declared"},
		Connector:           connectorCheck(resolve),
		Entrypoint:          facets.entrypoint,
		PermissionGrant:     grantCheck,
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
}

func requestedPermissions(root, provider string) []domain.PluginPermission {
	facts, err := domain.ResolveWeaponFacts(root, provider)
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

// probeSkillEntrypoint is the strongest entrypoint probe feasible for an
// external skill plugin from a static CLI check. A *true* live-invocation
// probe would mean actually running the skill through the Claude Code skill
// loader (a separate process this CLI does not control and cannot safely or
// deterministically invoke from `strategist check`), so this deliberately
// does not fake that — instead it verifies everything about the entrypoint
// manifest that a static check honestly can: the file this slot resolved to
// exists, is non-empty, is parseable YAML, and declares an `id` consistent
// with the provider it was resolved for. This replaces the previous
// unconditional `Entrypoint: Unsupported` hardcode (which never actually
// looked at the file) with a check that can and does return Blocked when the
// manifest is missing, empty, unparseable, or self-inconsistent.
func probeSkillEntrypoint(provider, path string) domain.ReadinessCheck {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_file_missing", Detail: path}
	}
	if info.Size() == 0 {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_file_empty", Detail: path}
	}
	raw, readErr := os.ReadFile(path) //nolint:gosec // G304: path is derived from the runtime skills directory
	if readErr != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_file_unreadable", Detail: readErr.Error()}
	}
	var manifest struct {
		ID string `yaml:"id"`
	}
	if yamlErr := yaml.Unmarshal(raw, &manifest); yamlErr != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_manifest_unparseable", Detail: yamlErr.Error()}
	}
	if manifest.ID == "" {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_id_missing", Detail: path}
	}
	if manifest.ID != provider {
		return domain.ReadinessCheck{
			Status:     domain.ReadinessBlocked,
			ReasonCode: "entrypoint_id_mismatch",
			Detail:     fmt.Sprintf("manifest id=%q provider=%q", manifest.ID, provider),
		}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "entrypoint_manifest_verified", Detail: path}
}

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

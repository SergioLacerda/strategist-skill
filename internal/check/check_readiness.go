package check

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

func skillProviderReadiness(root, slot, provider, path string) domain.PluginReadinessVector {
	connector := connectors.UnsupportedConnector{IDValue: "current-runtime", ConnectorAPIVersion: "strategist-connector-api/1"}
	resolve := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: provider, Path: path})
	observe := connector.Observe(context.Background(), domain.InstalledInstance{ID: provider})
	entrypoint := "refine"
	if slot == string(domain.SlotDiscovery) {
		entrypoint = "discover"
	}
	probe := connector.Probe(context.Background(), domain.InstalledInstance{ID: provider, ConnectorID: connector.Capabilities(context.Background()).ConnectorID}, entrypoint)
	lock := readPluginsLockFile(root)
	digest := lock.NodeDigest(provider, "adapter_contract")
	trustCheck := skillProviderTrustReadiness(root, provider, digest)
	grantCheck := skillProviderPermissionGrantReadinessFor(root, digest, requestedPermissions(path))
	ranked := bindingIsRanked(lock, slot, provider)
	conformance := customConformanceReadiness(root, slot, provider, path, probe)
	if ranked {
		// A Ranked binding is already validated and certified at build time
		// (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md
		// DEC-003) — it never calls into Custom's trust.Verify/
		// policy.EvaluateGrant runtime checks; readiness is reported from the
		// catalog's certification stamp instead.
		trustCheck, grantCheck = rankedCertificationReadiness(root, slot, provider)
		conformance = domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "ranked_certification_verified"}
	}
	dependencies := domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "dependency_lock_not_evaluated"}
	if !ranked {
		// A Ranked binding proves its runtime through its own readiness; every
		// other binding must still not read ready without one.
		dependencies = customRuntimeReadiness(root, slot, provider)
	}
	return domain.PluginReadinessVector{
		Descriptor:          domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "legacy_descriptor_valid", Detail: path},
		Source:              domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_manifest_present", Detail: path},
		Conformance:         conformance,
		Trust:               trustCheck,
		Dependencies:        dependencies,
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "host_api_not_declared"},
		Connector:           connectorCheck(resolve),
		Entrypoint:          probeSkillEntrypoint(provider, path),
		PermissionGrant:     grantCheck,
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
}

func requestedPermissions(path string) []domain.PluginPermission {
	raw, err := os.ReadFile(path) //nolint:gosec // path is the resolved runtime skill manifest
	if err != nil {
		return nil
	}
	var manifest struct {
		Requested []domain.PluginPermission `yaml:"requested_permissions"`
	}
	if yaml.Unmarshal(raw, &manifest) != nil {
		return nil
	}
	return manifest.Requested
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

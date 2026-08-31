package check

import (
	"context"
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

func skillProviderReadiness(provider, path string) domain.PluginReadinessVector {
	connector := connectors.UnsupportedConnector{IDValue: "current-runtime", ConnectorAPIVersion: "strategist-connector-api/1"}
	resolve := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: provider, Path: path})
	observe := connector.Observe(context.Background(), domain.InstalledInstance{ID: provider})
	return domain.PluginReadinessVector{
		Descriptor:          domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "legacy_descriptor_valid", Detail: path},
		Source:              domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "local_manifest_present", Detail: path},
		Trust:               domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "trust_policy_not_evaluated"},
		Dependencies:        domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "dependency_lock_not_evaluated"},
		HostAPI:             domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "host_api_not_declared"},
		Connector:           connectorCheck(resolve),
		Entrypoint:          probeSkillEntrypoint(provider, path),
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "permission_grant_not_evaluated"},
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
}

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

// readinessDimension names one PluginReadinessVector field for diagnostic
// messages, without requiring reflection or a change to the domain type.
type readinessDimension struct {
	name  string
	check domain.ReadinessCheck
}

func readinessDimensions(v domain.PluginReadinessVector) []readinessDimension {
	return []readinessDimension{
		{"descriptor", v.Descriptor},
		{"source", v.Source},
		{"trust", v.Trust},
		{"dependencies", v.Dependencies},
		{"host_api", v.HostAPI},
		{"connector", v.Connector},
		{"entrypoint", v.Entrypoint},
		{"permission_grant", v.PermissionGrant},
		{"enforcement_coverage", v.EnforcementCoverage},
		{"active_binding", v.ActiveBinding},
	}
}

// blockedReadinessErrors reports one message per dimension of slot's
// readiness vector that is explicitly domain.ReadinessBlocked — i.e. a
// dimension where the vector positively identifies an active problem, not
// merely one that is domain.ReadinessUnknown (not yet evaluated, by design —
// e.g. Trust/Dependencies/HostAPI/PermissionGrant are intentionally out of
// scope today) or domain.ReadinessUnsupported (the current runtime honestly
// does not offer that capability at all, e.g. an external skill plugin's
// Connector dimension). Gating strategist check's exit code on Blocked only
// — rather than on PluginReadinessVector.Ready(), which no configuration can
// satisfy today given those intentionally-unevaluated dimensions — means
// every currently-passing provider configuration keeps passing, while a
// configuration with a genuine, identifiable defect (e.g. an entrypoint
// manifest that doesn't exist or doesn't match its provider) newly fails
// with a precise slot+dimension+reason message instead of silently passing.
func blockedReadinessErrors(slot string, v domain.PluginReadinessVector) []string {
	var errs []string
	for _, d := range readinessDimensions(v) {
		if d.check.Status != domain.ReadinessBlocked {
			continue
		}
		msg := fmt.Sprintf("slot %s: readiness blocked on %s dimension (reason=%s", slot, d.name, d.check.ReasonCode)
		if d.check.Detail != "" {
			msg += ": " + d.check.Detail
		}
		msg += ")"
		errs = append(errs, msg)
	}
	return errs
}

// blockedReadinessErrorsForSlots applies blockedReadinessErrors across every
// slot in slots that has a resolution, collecting one error set. Extracted
// so check.go's RunE doesn't need to inline the per-slot loop itself.
func blockedReadinessErrorsForSlots(resolutions map[string]slotResolution, slots []string) []string {
	var errs []string
	for _, slot := range slots {
		res, ok := resolutions[slot]
		if !ok {
			continue
		}
		errs = append(errs, blockedReadinessErrors(slot, res.readiness)...)
	}
	return errs
}

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

package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

// slotContract maps slot names to their required risk_score contract.
var slotContract = map[string]string{
	"discovery":  "write_analysis",
	"refinement": "write_analysis",
	"execution":  "controlled",
}

// slotResolutionKind identifies which of the two independent resolver branches
// satisfied a slot: an external skill provider (skills/<provider>/skill.yaml,
// validated against risk_score) or a built-in Strategist native role
// (roles/<provider>.yaml, validated against RoleConfig.Validate + slot match).
// These are different authorities and must never be collapsed into one another
// — see .strategist/contracts/machine/preflight.yaml and design.md for
// 2026-07-25-native-role-resolution-check.
type slotResolutionKind string

const (
	slotResolutionSkillProvider slotResolutionKind = "skill_provider"
	slotResolutionNativeRole    slotResolutionKind = "native_role"
)

func (k slotResolutionKind) label() string {
	return domain.SlotExtensionKindLabel(string(k))
}

// slotResolution records how a slot's provider resolved and where its
// manifest/role definition lives, so callers (success table, --simulate
// report) can surface the resolution kind instead of just the provider id.
//
// fallbackProvider/fallbackPath are populated only for kind=skill_provider
// resolutions where a compatible native role also exists for the same slot
// (see resolveNativeFallback and docs/adr/0028-native-role-resilient-baseline.md).
// A native_role resolution never has a fallback — it already is the resilient
// baseline ADR-0028 describes.
type slotResolution struct {
	kind             slotResolutionKind
	path             string
	fallbackProvider string
	fallbackPath     string
	readiness        domain.PluginReadinessVector
}

// hasFallback reports whether a compatible native role was found for this
// slot resolution (always false for kind=native_role).
func (r slotResolution) hasFallback() bool {
	return r.fallbackProvider != ""
}

// resolveSlotProvider resolves provider for slot through the two-branch model:
// first as an external skill provider, then as a native Strategist role. On
// success it returns the resolution and an empty error message. On failure it
// returns a precise, branch-specific error message — a malformed or invalid
// native role file is never collapsed into a generic "provider not installed"
// message, since that would hide a real, fixable role-definition bug behind a
// message that reads as "nothing here at all".
func resolveSlotProvider(root, slot, provider string) (slotResolution, string) {
	skillPath := filepath.Join(root, "skills", provider, "skill.yaml")
	skillRaw, readErr := os.ReadFile(skillPath) //nolint:gosec // G304: provider manifest path is derived from the runtime skills directory
	if readErr == nil {
		return resolveSkillProviderSlot(slot, provider, skillPath, skillRaw)
	}
	if !os.IsNotExist(readErr) {
		return slotResolution{}, fmt.Sprintf("slot %s: read %s: %v", slot, skillPath, readErr)
	}
	return resolveNativeRoleSlot(root, slot, provider, skillPath)
}

func resolveSkillProviderSlot(slot, provider, skillPath string, skillRaw []byte) (slotResolution, string) {
	var skillDef struct {
		RiskScore string `yaml:"risk_score"`
	}
	if yamlErr := yaml.Unmarshal(skillRaw, &skillDef); yamlErr != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q skill.yaml invalid: %v", slot, provider, yamlErr)
	}
	required := slotContract[slot]
	if skillDef.RiskScore != required {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q has risk_score=%q but slot requires %q — preflight will block", slot, provider, skillDef.RiskScore, required)
	}
	return slotResolution{kind: slotResolutionSkillProvider, path: skillPath, readiness: skillProviderReadiness(provider, skillPath)}, ""
}

func resolveNativeRoleSlot(root, slot, provider, skillPath string) (slotResolution, string) {
	rolePath := filepath.Join(root, "roles", provider+".yaml")
	roleRaw, roleErr := os.ReadFile(rolePath) //nolint:gosec // G304: native role path is derived from the runtime roles directory
	if roleErr != nil {
		if os.IsNotExist(roleErr) {
			return slotResolution{}, fmt.Sprintf("slot %s: provider %q not installed (missing %s)", slot, provider, skillPath)
		}
		return slotResolution{}, fmt.Sprintf("slot %s: role %q unreadable: %v", slot, provider, roleErr)
	}
	var roleDef domain.RoleConfig
	if yamlErr := yaml.Unmarshal(roleRaw, &roleDef); yamlErr != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: role %q malformed YAML: %v", slot, provider, yamlErr)
	}
	if valErr := roleDef.Validate(); valErr != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: role %q invalid: %v", slot, provider, valErr)
	}
	if roleDef.Slot != slot {
		return slotResolution{}, fmt.Sprintf("slot %s: role %q declares slot=%q (mismatch)", slot, provider, roleDef.Slot)
	}
	return slotResolution{kind: slotResolutionNativeRole, path: rolePath, readiness: nativeRoleReadiness(provider, rolePath)}, ""
}

// resolveNativeFallback reports the compatible native role for slot, if one exists,
// so a skill_provider resolution can surface it as a fallback candidate (ADR-0028).
// The canonical slot→native-role mapping is roles/default.yaml (domain.RoleSlotMap);
// the candidate is only reported when resolveNativeRoleSlot independently validates
// it (existing role file, valid RoleConfig, matching slot) — the same validation
// already applied to any explicitly configured native-role provider, so a fallback
// is never offered for a role file that is missing, malformed, or slot-mismatched.
// Absence of roles/default.yaml, or any resolution error, is non-fatal: it simply
// means no fallback is reported, never a check failure — check's overall result for
// the slot is still governed entirely by the caller's own resolveSlotProvider outcome.
func resolveNativeFallback(root, slot string) (provider, path string) {
	defaultMapPath := filepath.Join(root, "roles", "default.yaml")
	raw, err := os.ReadFile(defaultMapPath) //nolint:gosec // G304: fixed path under the runtime roles directory
	if err != nil {
		return "", ""
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &roleMap); err != nil {
		return "", ""
	}
	candidate := roleMap[slot]
	if candidate == "" {
		return "", ""
	}
	res, errMsg := resolveNativeRoleSlot(root, slot, candidate, filepath.Join(root, "skills", candidate, "skill.yaml"))
	if errMsg != "" {
		return "", ""
	}
	return candidate, res.path
}

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
		Entrypoint:          domain.ReadinessCheck{Status: domain.ReadinessUnsupported, ReasonCode: "entrypoint_probe_unsupported"},
		PermissionGrant:     domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "permission_grant_not_evaluated"},
		EnforcementCoverage: connectorObservationCheck(observe),
		ActiveBinding:       domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "active_yaml_slot_binding"},
	}
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

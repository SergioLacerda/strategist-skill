package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
type slotResolution struct {
	kind      slotResolutionKind
	path      string
	readiness domain.PluginReadinessVector
	// transitionalView marks a Weapon resolved through a hand-made compat view the
	// catalog does not list (DEC-013): it stops resolving in the next runtime
	// layout generation, so the operator is told.
	transitionalView bool
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
	if res, msg, handled := resolveFromCatalog(root, slot, provider, skillPath); handled {
		return res, msg
	}
	skillRaw, readErr := os.ReadFile(skillPath) //nolint:gosec // G304: provider manifest path is derived from the runtime skills directory
	if readErr == nil {
		return resolveSkillProviderSlot(root, slot, provider, skillPath, skillRaw)
	}
	if !os.IsNotExist(readErr) {
		return slotResolution{}, fmt.Sprintf("slot %s: read %s: %v", slot, skillPath, readErr)
	}
	return resolveNativeRoleSlot(root, slot, provider, skillPath)
}

// resolveFromCatalog is the first step of slot resolution (DEC-010): a provider the
// catalog lists is resolved from its catalog entry, whether or not a generated
// compat view exists. A native_role entry takes the native branch, an embedded or
// external entry the Weapon branch. A provider the catalog does not list is not
// handled here and falls through to the transitional compat view and then to the
// native role file. A package added with `provider add` is not resolved by this
// path (U-01): DEC-010 step 2 is documented, not implemented.
func resolveFromCatalog(root, slot, provider, skillPath string) (slotResolution, string, bool) {
	facts, found, err := domain.ResolveCatalogWeaponFacts(root, provider)
	if err != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: plugin catalog invalid: %v", slot, err), true
	}
	if !found {
		return slotResolution{}, "", false
	}
	if facts.CompatibilitySource == "native_role" {
		res, msg := resolveNativeRoleSlot(root, slot, provider, skillPath)
		return res, msg, true
	}
	res, msg := resolveCatalogWeaponSlot(root, slot, provider, facts)
	return res, msg, true
}

func resolveSkillProviderSlot(root, slot, provider, skillPath string, skillRaw []byte) (slotResolution, string) {
	var skillDef struct {
		RiskScore string `yaml:"risk_score"`
	}
	// The compat view must still parse (entrypoint probing reads it); risk and
	// roles come from domain.ResolveWeaponFacts, where the catalog is the authority.
	if yamlErr := yaml.Unmarshal(skillRaw, &skillDef); yamlErr != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q skill.yaml invalid: %v", slot, provider, yamlErr)
	}
	facts, err := domain.ResolveWeaponFactsFrom(root, provider, skillRaw)
	if err != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q manifest unresolved: %v", slot, provider, err)
	}
	required := slotContract[slot]
	if facts.RiskScore != required {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q has risk_score=%q but slot requires %q — preflight will block", slot, provider, facts.RiskScore, required)
	}
	if errMsg := checkRoleFactsCompatibility(root, slot, provider, facts.RiskScore, facts.Roles); errMsg != "" {
		return slotResolution{}, errMsg
	}
	return slotResolution{kind: slotResolutionSkillProvider, path: skillPath, readiness: skillProviderReadiness(root, slot, provider, skillPath), transitionalView: true}, ""
}

// checkRoleProviderCompatibility lives in check_role_compatibility.go, split
// out to keep this file under the repo's file-size budget.

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

// loadRoleSlotMap reads roles/default.yaml — the canonical slot→native-role
// mapping used by role and weapon linkage verification.
func loadRoleSlotMap(root string) (domain.RoleSlotMap, error) {
	defaultMapPath := filepath.Join(root, "roles", "default.yaml")
	raw, err := os.ReadFile(defaultMapPath) //nolint:gosec // G304: fixed path under the runtime roles directory
	if err != nil {
		return nil, fmt.Errorf("read role slot map %s: %w", defaultMapPath, err)
	}
	var roleMap domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &roleMap); err != nil {
		return nil, fmt.Errorf("parse role slot map %s: %w", defaultMapPath, err)
	}
	return roleMap, nil
}

// Plugin-readiness vector computation (skillProviderReadiness,
// nativeRoleReadiness, probeSkillEntrypoint, blockedReadinessErrors, and
// their helpers) lives in check_readiness.go, split out to keep this file
// under the repo's file-size budget.

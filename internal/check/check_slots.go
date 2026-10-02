package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// slotContract maps slot names to their required risk_score contract; it is the
// list `provider add` also applies at bind time.
var slotContract = domain.SlotRiskContract

// slotResolutionKind identifies which of the independent resolver branches
// satisfied a slot: a catalog/custom Weapon or a built-in Strategist native role
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
}

// resolveSlotProvider resolves provider for slot through catalog/custom Weapon
// authority, then as a native Strategist role. On
// success it returns the resolution and an empty error message. On failure it
// returns a precise, branch-specific error message — a malformed or invalid
// native role file is never collapsed into a generic "provider not installed"
// message, since that would hide a real, fixable role-definition bug behind a
// message that reads as "nothing here at all".
func resolveSlotProvider(root, slot, provider string) (slotResolution, string) {
	if res, msg, handled := resolveFromCatalog(root, slot, provider); handled {
		return res, msg
	}
	if res, msg, handled := resolveFromCustomBinding(root, slot, provider); handled {
		return res, msg
	}
	return resolveNativeRoleSlot(root, slot, provider)
}

// resolveFromCatalog is the first step of slot resolution (DEC-010): a provider the
// catalog lists is resolved from its catalog entry. A native_role entry takes the
// native branch, an embedded or
// external entry the Weapon branch. A provider the catalog does not list is not
// handled here and falls through to the native role file. A package added with
// `provider add` is resolved by the next step, resolveFromCustomBinding (DEC-010
// step 2).
func resolveFromCatalog(root, slot, provider string) (slotResolution, string, bool) {
	facts, found, err := domain.ResolveCatalogWeaponFacts(root, provider)
	if err != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: plugin catalog invalid: %v", slot, err), true
	}
	if !found {
		return slotResolution{}, "", false
	}
	if facts.CompatibilitySource == "native_role" {
		res, msg := resolveNativeRoleSlot(root, slot, provider)
		return res, msg, true
	}
	res, msg := resolveCatalogWeaponSlot(root, slot, provider, facts)
	return res, msg, true
}

// checkRoleProviderCompatibility lives in check_role_compatibility.go, split
// out to keep this file under the repo's file-size budget.

func resolveNativeRoleSlot(root, slot, provider string) (slotResolution, string) {
	rolePath := filepath.Join(root, "roles", provider+".yaml")
	roleRaw, roleErr := os.ReadFile(rolePath) //nolint:gosec // G304: native role path is derived from the runtime roles directory
	if roleErr != nil {
		if os.IsNotExist(roleErr) {
			return slotResolution{}, fmt.Sprintf("slot %s: provider %q not installed (missing catalog/custom binding or %s)", slot, provider, rolePath)
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

// Plugin-readiness vector computation (nativeRoleReadiness, blockedReadinessErrors, and
// their helpers) lives in check_readiness.go, split out to keep this file
// under the repo's file-size budget.

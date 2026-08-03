package main

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

// slotResolution records how a slot's provider resolved and where its
// manifest/role definition lives, so callers (success table, --simulate
// report) can surface the resolution kind instead of just the provider id.
type slotResolution struct {
	kind slotResolutionKind
	path string
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
	return slotResolution{kind: slotResolutionSkillProvider, path: skillPath}, ""
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
	return slotResolution{kind: slotResolutionNativeRole, path: rolePath}, ""
}

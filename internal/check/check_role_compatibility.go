package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// checkRoleProviderCompatibility exercises the real
// domain.ProviderContract.CheckRoleAffinity algorithm for a configured
// skill_provider slot resolution — closing the gap where RoleContract/
// ProviderContract (internal/domain/role_provider_contract.go) and
// ResolveRoleBinding (internal/plugins/role_binding.go) existed, fully
// tested, with zero production callers (see .analysis/pending/skills_plugaveis/
// 20260913-role-provider-convergence-evaluation/analysis.md KF-05/KF-07).
//
// It runs the same role-affinity and role-contract-version dimensions used by
// the Wizard. Handoff production is owned by the fixed role checkpoint, so a
// weapon is not filtered merely because its own manifest omits a handoff
// schema declaration.
//
// It is a no-op (returns "") whenever:
//   - roles/default.yaml has no entry for slot, is unreadable, or the mapped
//     role's own role file is unreadable/invalid — those conditions are
//     already surfaced elsewhere (check_weapon_bindings.go)
//     and are not duplicated here;
//   - the skill.yaml declares no canonical_role at all — not every skill
//     provider is expected to declare one (e.g. sdd-ask), and the execution
//     slot's embedded weapon is explicitly deferred by
//     the fixed embedded-weapon roster.
//
// SupportedRoleContractVersions is not read from skill.yaml — no shipped
// manifest declares it yet — and is instead defaulted to exactly the
// resolved RoleContract's own SchemaVersion, so an existing manifest that
// predates this field is treated as compatible-by-default rather than
// spuriously rejected; only an actual canonical_role mismatch fails.
// The catalog handoff declaration remains available for full compatibility
// checks, but it is intentionally not part of this role-selection gate.
func checkRoleProviderCompatibility(root, slot, provider, riskScore string, skillRaw []byte) string {
	roles, err := loadProviderRoles(skillRaw)
	if err != nil {
		return ""
	}
	return checkRoleFactsCompatibility(root, slot, provider, riskScore, roles)
}

// checkRoleFactsCompatibility is the affinity check over roles already resolved
// from the Weapon's manifest (catalog first, compat view as fallback).
func checkRoleFactsCompatibility(root, slot, provider, riskScore string, roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	roleSlotMap, err := loadRoleSlotMap(root)
	if err != nil {
		return ""
	}
	roleID := roleSlotMap[slot]
	if roleID == "" {
		return ""
	}
	roleCfg, err := loadCompatibleRole(root, roleID)
	if err != nil {
		return ""
	}
	roleContract := domain.RoleContractFromConfig(roleCfg, "")
	providerContract := domain.ProviderContract{
		SchemaVersion:                 roleContract.SchemaVersion,
		ID:                            provider,
		Version:                       "0.0.0",
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 roles[0],
		Roles:                         roles,
		RiskScore:                     riskScore,
		Source:                        domain.ProviderSourceExternal,
		SupportedRoleContractVersions: []string{roleContract.SchemaVersion},
	}
	result := providerContract.CheckRoleAffinity(roleContract)
	if result.Compatible {
		return ""
	}
	return formatRoleCompatibilityFailure(slot, provider, roleID, result)
}

func loadCompatibleRole(root, roleID string) (domain.RoleConfig, error) {
	roleRaw, err := os.ReadFile(filepath.Join(root, "roles", roleID+".yaml")) //nolint:gosec // G304: path derived from the runtime roles directory
	if err != nil {
		return domain.RoleConfig{}, fmt.Errorf("read role %s: %w", roleID, err)
	}
	var roleCfg domain.RoleConfig
	if err := yaml.Unmarshal(roleRaw, &roleCfg); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("parse role %s: %w", roleID, err)
	}
	if err := roleCfg.Validate(); err != nil {
		return domain.RoleConfig{}, fmt.Errorf("validate role %s: %w", roleID, err)
	}
	return roleCfg, nil
}

func loadProviderRoles(skillRaw []byte) ([]string, error) {
	var taxonomy skillTaxonomy
	if err := yaml.Unmarshal(skillRaw, &taxonomy); err != nil {
		return nil, fmt.Errorf("parse provider taxonomy: %w", err)
	}
	return taxonomy.roles(), nil
}

func formatRoleCompatibilityFailure(slot, provider, roleID string, result domain.CompatibilityResult) string {
	details := make([]string, 0, len(result.Reasons))
	for _, reason := range result.Reasons {
		details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
	}
	return fmt.Sprintf("slot %s: provider %q role-incompatible with %q: %s", slot, provider, roleID, strings.Join(details, "; "))
}

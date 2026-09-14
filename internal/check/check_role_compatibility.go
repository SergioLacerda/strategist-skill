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
// domain.ProviderContract.CheckRoleCompatibility algorithm for a configured
// skill_provider slot resolution — closing the gap where RoleContract/
// ProviderContract (internal/domain/role_provider_contract.go) and
// ResolveRoleBinding (internal/plugins/role_binding.go) existed, fully
// tested, with zero production callers (see .analysis/pending/skills_plugaveis/
// 20260913-role-provider-convergence-evaluation/analysis.md KF-05/KF-07).
//
// Scope is deliberately narrow: it only runs the canonical_role dimension.
// It is a no-op (returns "") whenever:
//   - roles/default.yaml has no entry for slot, is unreadable, or the mapped
//     role's own role file is unreadable/invalid — those conditions are
//     already surfaced elsewhere (resolveNativeFallback, check_weapon_bindings.go)
//     and are not duplicated here;
//   - the skill.yaml declares no canonical_role at all — not every skill
//     provider is expected to declare one (e.g. sdd-ask), and the execution
//     slot's embedded weapon is explicitly deferred by
//     docs/adr/0035-embedded-weapon-fallback-policy.md DEC-001.
//
// SupportedRoleContractVersions is not read from skill.yaml — no shipped
// manifest declares it yet — and is instead defaulted to exactly the
// resolved RoleContract's own SchemaVersion, so an existing manifest that
// predates this field is treated as compatible-by-default rather than
// spuriously rejected; only an actual canonical_role mismatch fails.
func checkRoleProviderCompatibility(root, slot, provider, riskScore string, skillRaw []byte) string {
	roleSlotMap, err := loadRoleSlotMap(root)
	if err != nil {
		return ""
	}
	roleID := roleSlotMap[slot]
	if roleID == "" {
		return ""
	}
	roleRaw, err := os.ReadFile(filepath.Join(root, "roles", roleID+".yaml")) //nolint:gosec // G304: path derived from the runtime roles directory
	if err != nil {
		return ""
	}
	var roleCfg domain.RoleConfig
	if yaml.Unmarshal(roleRaw, &roleCfg) != nil || roleCfg.Validate() != nil {
		return ""
	}

	var taxonomy skillTaxonomy
	if yaml.Unmarshal(skillRaw, &taxonomy) != nil {
		return ""
	}
	canonicalRole := taxonomy.canonicalRole()
	if canonicalRole == "" {
		return ""
	}

	roleContract := domain.RoleContractFromConfig(roleCfg, "")
	providerContract := domain.ProviderContract{
		SchemaVersion:                 roleContract.SchemaVersion,
		ID:                            provider,
		Version:                       "0.0.0",
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 canonicalRole,
		RiskScore:                     riskScore,
		Source:                        domain.ProviderSourceExternal,
		SupportedRoleContractVersions: []string{roleContract.SchemaVersion},
	}
	result := providerContract.CheckRoleCompatibility(roleContract)
	if result.Compatible {
		return ""
	}
	details := make([]string, 0, len(result.Reasons))
	for _, reason := range result.Reasons {
		details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
	}
	return fmt.Sprintf("slot %s: provider %q role-incompatible with %q: %s", slot, provider, roleID, strings.Join(details, "; "))
}

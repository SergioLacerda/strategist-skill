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
// It runs the same role-affinity and role-contract-version dimensions used by
// the Wizard. Handoff production is owned by the fixed role checkpoint, so a
// weapon is not filtered merely because its own manifest omits a handoff
// schema declaration.
//
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
// The catalog handoff declaration remains available for full compatibility
// checks, but it is intentionally not part of this role-selection gate.
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
	roles := taxonomy.roles()
	if len(roles) == 0 {
		return ""
	}

	handoffSchema, cataloged := loadSupportedHandoffSchemas(root, provider)
	roleContract := domain.RoleContractFromConfig(roleCfg, "")
	if cataloged {
		roleContract = domain.RoleContractFromConfig(roleCfg, domain.RoleHandoffSchema[roleID])
	}
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
		SupportedHandoffSchemas:       handoffSchema,
	}
	result := providerContract.CheckRoleAffinity(roleContract)
	if result.Compatible {
		return ""
	}
	details := make([]string, 0, len(result.Reasons))
	for _, reason := range result.Reasons {
		details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
	}
	return fmt.Sprintf("slot %s: provider %q role-incompatible with %q: %s", slot, provider, roleID, strings.Join(details, "; "))
}

// loadSupportedHandoffSchemas reads root/plugins/catalog.yaml — the same
// file internal/install's Wizard reads via loadPluginCatalog — and returns
// the named provider's own declared supported_handoff_schemas, or nil when
// the catalog is absent/unreadable, the provider has no entry, or the entry
// declares none. A read/parse failure is treated the same as "declares
// none" (fail-closed on this dimension only) rather than a check error,
// consistent with this file's other lookups (loadRoleSlotMap, role file
// reads) that degrade to "no opinion" on I/O failure.
func loadSupportedHandoffSchemas(root, provider string) ([]string, bool) {
	catalogPath := filepath.Join(root, "plugins", "catalog.yaml")
	raw, err := os.ReadFile(catalogPath) //nolint:gosec // G304: fixed path under the runtime plugins directory
	if err != nil {
		return nil, false
	}
	var doc struct {
		Providers []struct {
			ID                      string   `yaml:"id"`
			SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas"`
		} `yaml:"providers"`
	}
	if yaml.Unmarshal(raw, &doc) != nil {
		return nil, false
	}
	for _, p := range doc.Providers {
		if p.ID == provider {
			return p.SupportedHandoffSchemas, true
		}
	}
	return nil, false
}

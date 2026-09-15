package domain

import "fmt"

// CheckRoleCompatibility evaluates whether this Provider may bind to role.
// This is the canonical_role/role-contract-version compatibility dimension
// required by proposal.md Decision 5, additive to
// AdapterContract.CheckCompatibility's existing host-API dimension — it does
// not replace it.
func (p ProviderContract) CheckRoleCompatibility(role RoleContract) CompatibilityResult {
	affinity := p.CheckRoleAffinity(role)
	if !affinity.Compatible {
		return affinity
	}
	if p.Source != ProviderSourceNativeRole && role.HandoffSchema != "" &&
		!hasString(stringSet(p.SupportedHandoffSchemas...), role.HandoffSchema) {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "handoff_schema",
			Code:      "unsupported_handoff_schema",
			Detail:    fmt.Sprintf("%s does not declare support for handoff schema %s", p.ID, role.HandoffSchema),
		}}}
	}
	return CompatibilityResult{Compatible: true}
}

// CheckRoleAffinity verifies only the provider's explicit affinity and role
// contract version. Handoff production is a fixed role checkpoint concern,
// so this method is used by catalogs and wizard selection before execution.
func (p ProviderContract) CheckRoleAffinity(role RoleContract) CompatibilityResult {
	roles := p.Roles
	if len(roles) == 0 && p.CanonicalRole != "" {
		roles = []string{p.CanonicalRole}
	}
	if !hasString(stringSet(roles...), role.Role) {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "role_affinity",
			Code:      "role_mismatch",
			Detail:    fmt.Sprintf("provider declares roles %v, role contract is %q", roles, role.Role),
		}}}
	}
	if !hasString(stringSet(p.SupportedRoleContractVersions...), role.SchemaVersion) {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "role_contract_version",
			Code:      "unsupported_role_contract_version",
			Detail:    fmt.Sprintf("%s does not declare support for role contract %s", p.ID, role.SchemaVersion),
		}}}
	}
	return CompatibilityResult{Compatible: true}
}

// ProviderBinding is the resolved Role -> Provider association at the
// compatibility-view layer. Workspace-local persistence of the effective
// binding remains owned by SlotBinding (see PluginResourceBinding
// authority in plugin_types.go) — ProviderBinding composes a RoleContract
// and ProviderContract with the resulting CompatibilityResult without
// creating a second binding store.
type ProviderBinding struct {
	Role          RoleContract
	Provider      ProviderContract
	Compatibility CompatibilityResult
}

// ResolveProviderBinding computes the ProviderBinding for role and provider
// by running CheckRoleCompatibility. It does not persist anything —
// persistence stays with SlotBinding per the existing
// PluginResourceBinding authority.
func ResolveProviderBinding(role RoleContract, provider ProviderContract) ProviderBinding {
	return ProviderBinding{
		Role:          role,
		Provider:      provider,
		Compatibility: provider.CheckRoleCompatibility(role),
	}
}

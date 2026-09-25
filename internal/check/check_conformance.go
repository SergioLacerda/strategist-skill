package check

import (
	"fmt"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/conformance"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// customConformanceReadiness evaluates the static contract evidence available
// for a Custom/non-Ranked skill. It intentionally does not claim runtime
// invocation: that evidence belongs to the connector dimension and remains
// unknown/unsupported when the host cannot provide it.
func customConformanceReadiness(root, slot, provider, path string, probe connectors.ConnectorResult) domain.ReadinessCheck {
	return customConformanceReadinessFor(root, slot, provider, func() ([]string, domain.ReadinessCheck) { return rolesFromView(path) }, probe)
}

// customConformanceReadinessFor is the conformance evidence for a Weapon; rolesOf
// supplies its role affinity (from the catalog or the compat view) and is called
// only after the slot's own role contract resolved, so the failure order is stable.
func customConformanceReadinessFor(root, slot, provider string, rolesOf func() ([]string, domain.ReadinessCheck), probe connectors.ConnectorResult) domain.ReadinessCheck {
	roleID, roleContract, failure := resolveCustomRole(root, slot)
	if failure.Status != "" {
		return failure
	}
	roles, failure := rolesOf()
	if failure.Status != "" {
		return failure
	}
	providerContract := customProviderContract(provider, slot, roleContract.SchemaVersion, roles)
	result := providerContract.CheckRoleAffinity(roleContract)
	if !result.Compatible {
		return conformanceCheck(domain.ReadinessBlocked, "conformance_role_mismatch", formatRoleCompatibilityFailure(slot, provider, roleID, result))
	}
	state := conformance.StateForReadiness(probe.Status)
	reason := probe.ReasonCode
	if reason == "" {
		reason = "conformance_probe_" + string(state)
	}
	return conformanceCheck(probe.Status, reason,
		fmt.Sprintf("state=%s role=%s provider=%s probe=%s", state, roleID, provider, probe.Detail))
}

func resolveCustomRole(root, slot string) (string, domain.RoleContract, domain.ReadinessCheck) {
	roleMap, err := loadRoleSlotMap(root)
	if err != nil {
		return "", domain.RoleContract{}, conformanceCheck(domain.ReadinessUnknown, "conformance_role_mapping_unknown", err.Error())
	}
	roleID := roleMap[slot]
	if roleID == "" {
		return "", domain.RoleContract{}, conformanceCheck(domain.ReadinessUnknown, "conformance_role_not_mapped", slot)
	}
	roleCfg, err := loadCompatibleRole(root, roleID)
	if err != nil {
		return "", domain.RoleContract{}, conformanceCheck(domain.ReadinessBlocked, "conformance_role_contract_invalid", err.Error())
	}
	return roleID, domain.RoleContractFromConfig(roleCfg, ""), domain.ReadinessCheck{}
}

func rolesFromView(path string) ([]string, domain.ReadinessCheck) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the selected runtime provider
	if err != nil {
		return nil, conformanceCheck(domain.ReadinessUnknown, "conformance_provider_manifest_unknown", err.Error())
	}
	roles, err := loadProviderRoles(raw)
	if err != nil || len(roles) == 0 {
		return nil, conformanceCheck(domain.ReadinessUnknown, "conformance_role_affinity_unknown", "provider does not declare canonical_role or roles")
	}
	return roles, domain.ReadinessCheck{}
}

func customProviderContract(provider, slot, roleSchema string, roles []string) domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 roleSchema,
		ID:                            provider,
		Version:                       "0.0.0",
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 roles[0],
		Roles:                         roles,
		RiskScore:                     slotContract[slot],
		Source:                        domain.ProviderSourceExternal,
		SupportedRoleContractVersions: []string{roleSchema},
	}
}

func conformanceCheck(status domain.ReadinessStatus, reason, detail string) domain.ReadinessCheck {
	return domain.ReadinessCheck{
		Status:        status,
		EvidenceState: string(conformance.StateForReadiness(status)),
		ReasonCode:    reason,
		Detail:        detail,
	}
}

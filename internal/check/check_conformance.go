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
	roleID, roleContract, failure := resolveCustomRole(root, slot)
	if failure.Status != "" {
		return failure
	}
	providerContract, failure := resolveCustomProvider(path, provider, slot, roleContract.SchemaVersion)
	if failure.Status != "" {
		return failure
	}
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

func resolveCustomProvider(path, provider, slot, roleSchema string) (domain.ProviderContract, domain.ReadinessCheck) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the selected runtime provider
	if err != nil {
		return domain.ProviderContract{}, conformanceCheck(domain.ReadinessUnknown, "conformance_provider_manifest_unknown", err.Error())
	}
	roles, err := loadProviderRoles(raw)
	if err != nil || len(roles) == 0 {
		return domain.ProviderContract{}, conformanceCheck(domain.ReadinessUnknown, "conformance_role_affinity_unknown", "provider does not declare canonical_role or roles")
	}
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
	}, domain.ReadinessCheck{}
}

func conformanceCheck(status domain.ReadinessStatus, reason, detail string) domain.ReadinessCheck {
	return domain.ReadinessCheck{
		Status:        status,
		EvidenceState: string(conformance.StateForReadiness(status)),
		ReasonCode:    reason,
		Detail:        detail,
	}
}

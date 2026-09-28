package provider

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// executionRiskFloor is the risk_score the execution slot requires, from the same
// slot contract `strategist check` enforces.
var executionRiskFloor = domain.SlotRiskContract[string(domain.SlotExecution)]

// documentationOnlyPermissions is the widest permission set a custom Sniper may
// request. Sniper materializes approved documentation targets only, so any
// permission that reaches source, network, subprocesses, secrets or external
// apps is refused at bind time instead of being discovered at mission time.
var documentationOnlyPermissions = map[domain.PluginPermission]bool{
	domain.PluginPermissionReadWorkspace: true,
	domain.PluginPermissionWriteAnalysis: true,
	domain.PluginPermissionWriteDocs:     true,
}

// validateExecutionBinding applies the extra bind-time rules for a custom
// Weapon on the Sniper role (ADR-0034 amendment: Sniper is pluggable, the ranked
// binding is the default, and a custom Weapon is accepted only when it cannot
// widen Sniper's documentation-only authority). Other slots are unaffected.
func validateExecutionBinding(source Source, requestedSlot string) []Reason {
	if requestedSlot != string(domain.SlotExecution) {
		return nil
	}
	var reasons []Reason
	if !contains(source.Adapter.SupportedRoles, sniperRoleID) {
		reasons = append(reasons, Reason{Code: "execution_role_affinity_missing", Detail: fmt.Sprintf("adapter must declare supported_roles including %q to bind the execution slot", sniperRoleID)})
	}
	reasons = append(reasons, executionRiskReasons(source)...)
	return append(reasons, executionPermissionReasons(source.Adapter.RequestedPermissions)...)
}

// executionRiskReasons reports a declared risk_score below the execution
// contract. Risk is optional; its absence is left to `strategist check`, which
// reads the same slot contract at mission time.
func executionRiskReasons(source Source) []Reason {
	risk, where := declaredRisk(source)
	if risk == "" || risk == executionRiskFloor {
		return nil
	}
	return []Reason{{Code: "execution_risk_below_controlled", Detail: fmt.Sprintf("%s risk_score=%q, the execution slot requires %q", where, risk, executionRiskFloor)}}
}

// validateAnalysisBinding applies the bind-time risk rule for the discovery and
// refinement slots. `strategist check` reads a custom package's risk_score from its
// adapter.yaml only and blocks the slot unless it equals the slot contract, so a
// package that declares none, or another value, is refused here instead of being
// accepted and then blocked at mission time. The compat view is not consulted.
func validateAnalysisBinding(source Source, requestedSlot string) []Reason {
	required := domain.SlotRiskContract[requestedSlot]
	if requestedSlot == string(domain.SlotExecution) || required == "" {
		return nil
	}
	switch declared := source.Adapter.RiskScore; declared {
	case required:
		return nil
	case "":
		return []Reason{{Code: "analysis_risk_missing", Detail: fmt.Sprintf("adapter.yaml must declare risk_score %q to bind the %s slot", required, requestedSlot)}}
	default:
		return []Reason{{Code: "analysis_risk_mismatch", Detail: fmt.Sprintf("adapter.yaml risk_score=%q, the %s slot requires %q", declared, requestedSlot, required)}}
	}
}

// declaredRisk returns the risk the source declares and where it declared it.
// The adapter is the authority; the compat skill.yaml view is read only when the
// adapter declares none.
func declaredRisk(source Source) (risk, where string) {
	if source.Adapter.RiskScore != "" {
		return source.Adapter.RiskScore, "adapter.yaml"
	}
	if source.Legacy != nil {
		return source.Legacy.RiskScore, "skill.yaml"
	}
	return "", ""
}

func executionPermissionReasons(requested []domain.PluginPermission) []Reason {
	var reasons []Reason
	for _, permission := range requested {
		if !documentationOnlyPermissions[permission] {
			reasons = append(reasons, Reason{Code: "execution_permission_exceeds_documentation", Detail: string(permission)})
		}
	}
	return reasons
}

const sniperRoleID = "sniper"

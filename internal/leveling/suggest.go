package leveling

import (
	"fmt"
	"strings"
)

// Suggest resolves a role recommendation and translates it for a ranked provider.
func Suggest(policy Policy, provider, role string, signals Signals) (Suggestion, error) {
	if err := policy.Validate(); err != nil {
		return Suggestion{}, err
	}
	if err := validateSignals(signals); err != nil {
		return Suggestion{}, err
	}
	providerID := strings.ToUpper(strings.TrimSpace(provider))
	if providerID == "" {
		return Suggestion{}, fmt.Errorf("leveling_ranked_provider_ineligible: provider identifier must not be empty")
	}
	roleID := strings.ToLower(strings.TrimSpace(role))
	rolePolicy := policyRole(policy, roleID)
	capability, effort, rationale := roleRecommendation(rolePolicy, signals)
	profile, found := policy.Providers[providerID]
	result := Suggestion{Provider: providerID, Role: roleID, Capability: capability, Effort: effort, Rationale: rationale, PolicyVersion: policy.Version, PolicyDigest: policy.Digest()}
	if !found {
		result.FallbackUsed = true
		result.FallbackReason = policy.Defaults.Fallback.Reason
		result.Rationale = append(result.Rationale, "generic_provider_fallback")
		return result, nil
	}
	if !profile.Ranked {
		return Suggestion{}, fmt.Errorf("leveling_ranked_provider_ineligible: provider %q is not ranked", providerID)
	}
	if !contains(profile.EffortTiers, effort) {
		return Suggestion{}, fmt.Errorf("leveling_ranked_provider_ineligible: provider %q cannot execute effort %q for role %q", providerID, effort, roleID)
	}
	model := profile.Models[capability]
	if model == "" {
		return Suggestion{}, fmt.Errorf("leveling_mapping_invalid: provider %q has no model mapping for capability %q", providerID, capability)
	}
	result.Model = model
	return result, nil
}

func validateSignals(signals Signals) error {
	checks := []struct {
		name  string
		value string
		valid []string
	}{
		{"ambiguity", signals.Ambiguity, []string{"low", "medium", "high"}},
		{"risk", signals.Risk, []string{"low", "medium", "high"}},
		{"scope", signals.Scope, []string{"bounded", "cross_module"}},
		{"evidence", signals.Evidence, []string{"sufficient", "insufficient", "conflicting"}},
	}
	for _, check := range checks {
		if check.value != "" && !contains(check.valid, check.value) {
			return fmt.Errorf("leveling_signal_unknown: %s=%q is not supported", check.name, check.value)
		}
	}
	if signals.RepeatedFailures < 0 {
		return fmt.Errorf("leveling_signal_unknown: repeated_failures must not be negative")
	}
	return nil
}

func policyRole(policy Policy, roleID string) Role {
	if role, ok := policy.Defaults.Roles[roleID]; ok {
		return role
	}
	return Role{Capability: policy.Defaults.Fallback.Capability, Effort: policy.Defaults.Fallback.Effort}
}

func roleRecommendation(role Role, signals Signals) (string, string, []string) {
	capability, effort := role.Capability, role.Effort
	rationale := []string{"generic_role_defaults"}
	if shouldEscalate(signals, role.Criteria) && role.Escalation.Capability != "" {
		capability, effort = role.Escalation.Capability, role.Escalation.Effort
		rationale = append(rationale, escalationReason(signals))
	}
	return capability, effort, rationale
}

func shouldEscalate(signals Signals, criteria Criteria) bool {
	return signals.ArchitecturalChange || signals.SecuritySensitive || signals.ConflictingEvidence || signals.RepeatedFailures >= 2 ||
		exceedsLevel(signals.Ambiguity, criteria.Ambiguity) || exceedsLevel(signals.Risk, criteria.Risk) ||
		exceedsScope(signals.Scope, criteria.Scope) || exceedsEvidence(signals.Evidence, criteria.Evidence)
}

func exceedsLevel(signal, baseline string) bool {
	levels := map[string]int{"low": 1, "medium": 2, "high": 3}
	if signal == "" {
		return false
	}
	if baseline == "" {
		return levels[signal] >= 3
	}
	return levels[signal] > levels[baseline]
}

func exceedsScope(signal, baseline string) bool {
	return signal == "cross_module" && baseline != "cross_module"
}

func exceedsEvidence(signal, baseline string) bool {
	if signal == "" {
		return false
	}
	if signal == "conflicting" {
		return true
	}
	return signal == "insufficient" && baseline == "sufficient"
}

func escalationReason(signals Signals) string {
	switch {
	case signals.SecuritySensitive:
		return "security_sensitive"
	case signals.ArchitecturalChange:
		return "architectural_change"
	case signals.ConflictingEvidence || signals.Evidence == "conflicting":
		return "conflicting_evidence"
	case signals.RepeatedFailures >= 2:
		return "repeated_failure"
	case signals.Ambiguity == "high":
		return "high_ambiguity"
	case signals.Risk == "high":
		return "high_risk"
	case signals.Scope == "cross_module":
		return "cross_module_scope"
	default:
		return "insufficient_evidence"
	}
}

package leveling

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Validate rejects incomplete or unsafe LEVELING policy documents.
func (p Policy) Validate() error {
	if p.Version <= 0 {
		return fmt.Errorf("leveling: version must be positive")
	}
	if err := validateDefaults(p.Defaults); err != nil {
		return err
	}
	if err := validateRoles(p.Defaults.Roles, p.Defaults.EffortTiers); err != nil {
		return err
	}
	if err := validateProviders(p.Providers); err != nil {
		return err
	}
	return validateCapabilityMappings(p.Defaults, p.Providers)
}

func validateDefaults(defaults Defaults) error {
	if err := validateEffortTiers("defaults.effort_tiers", defaults.EffortTiers); err != nil {
		return err
	}
	return validateDefaultFallback(defaults)
}

func validateDefaultFallback(defaults Defaults) error {
	if err := validateCapability("defaults.fallback.capability", defaults.Fallback.Capability); err != nil {
		return err
	}
	if strings.TrimSpace(defaults.Fallback.Effort) == "" || strings.TrimSpace(defaults.Fallback.Reason) == "" {
		return fmt.Errorf("leveling: defaults.fallback requires capability, effort, and reason")
	}
	if !contains(defaults.EffortTiers, defaults.Fallback.Effort) {
		return fmt.Errorf("leveling_mapping_invalid: defaults.fallback.effort %q is not declared", defaults.Fallback.Effort)
	}
	return validateOptionalBudget("defaults.budget", defaults.Budget)
}

func validateOptionalBudget(path string, budget domain.EffortBudgetPolicy) error {
	if budgetPolicyEmpty(budget) {
		return nil
	}
	return validateBudgetPolicy(path, budget)
}

func validateRoles(roles map[string]Role, tiers []string) error {
	for roleID, role := range roles {
		if err := validateRole(roleID, role, tiers); err != nil {
			return err
		}
	}
	return nil
}

func validateProviders(providers map[string]Provider) error {
	if len(providers) == 0 {
		return fmt.Errorf("leveling: at least one provider profile is required")
	}
	for id, provider := range providers {
		if err := validateProvider(id, provider); err != nil {
			return err
		}
	}
	return nil
}

func validateRole(id string, role Role, tiers []string) error {
	if err := validateRoleBasics(id, role, tiers); err != nil {
		return err
	}
	if err := validateCriteria(id, role.Criteria); err != nil {
		return err
	}
	if !budgetPolicyEmpty(role.Budget) {
		if err := validateBudgetPolicy("defaults.roles."+id+".budget", role.Budget); err != nil {
			return err
		}
	}
	return validateRoleEscalation(id, role.Escalation, tiers)
}

func budgetPolicyEmpty(policy domain.EffortBudgetPolicy) bool {
	return policy.MaxDelegations == 0 && policy.MaxDepth == 0 && policy.Parallelism == 0 &&
		policy.ContextBudget == "" && policy.EvidenceDepth == "" &&
		!policy.Permissions.Specialists && !policy.Permissions.SideQuests && !policy.Permissions.DeepHistory &&
		!policy.EscalationAllowed && policy.MaxEscalations == 0
}

func validateBudgetPolicy(path string, policy domain.EffortBudgetPolicy) error {
	if policy.MaxDelegations < 0 || policy.MaxDepth < 0 || policy.MaxEscalations < 0 {
		return fmt.Errorf("%s: %s: limits must not be negative", ReasonLevelingBudgetInvalid, path)
	}
	if policy.Parallelism < 1 {
		return fmt.Errorf("%s: %s.parallelism must be positive", ReasonLevelingBudgetInvalid, path)
	}
	if !contains([]string{"low", "medium", "high"}, strings.ToLower(strings.TrimSpace(policy.ContextBudget))) {
		return fmt.Errorf("%s: %s.context_budget %q is invalid", ReasonLevelingBudgetInvalid, path, policy.ContextBudget)
	}
	if !contains([]string{"minimal", "standard", "deep"}, strings.ToLower(strings.TrimSpace(policy.EvidenceDepth))) {
		return fmt.Errorf("%s: %s.evidence_depth %q is invalid", ReasonLevelingBudgetInvalid, path, policy.EvidenceDepth)
	}
	if !policy.EscalationAllowed && policy.MaxEscalations != 0 {
		return fmt.Errorf("%s: %s.max_escalations must be zero when escalation is disabled", ReasonLevelingBudgetInvalid, path)
	}
	return nil
}

func validateRoleBasics(id string, role Role, tiers []string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("leveling: defaults.roles.%s requires a non-empty identifier", id)
	}
	if err := validateCapability("defaults.roles."+id+".capability", role.Capability); err != nil {
		return err
	}
	if strings.TrimSpace(role.Effort) == "" {
		return fmt.Errorf("leveling: defaults.roles.%s requires capability and effort", id)
	}
	if !contains(tiers, role.Effort) {
		return fmt.Errorf("leveling_mapping_invalid: defaults.roles.%s.effort %q is not declared", id, role.Effort)
	}
	return nil
}

func validateRoleEscalation(id string, escalation Escalation, tiers []string) error {
	if err := validateEscalationCapability(id, escalation.Capability); err != nil {
		return err
	}
	return validateEscalationEffort(id, escalation, tiers)
}

func validateEscalationCapability(id, capability string) error {
	if capability == "" {
		return nil
	}
	return validateCapability("defaults.roles."+id+".escalation.capability", capability)
}

func validateEscalationEffort(id string, escalation Escalation, tiers []string) error {
	if escalation.Capability != "" && !contains(tiers, escalation.Effort) {
		return fmt.Errorf("leveling_mapping_invalid: defaults.roles.%s.escalation.effort %q is not declared", id, escalation.Effort)
	}
	if escalation.Effort != "" && strings.TrimSpace(escalation.Capability) == "" {
		return fmt.Errorf("leveling: defaults.roles.%s.escalation requires capability", id)
	}
	return nil
}

func validateCriteria(id string, criteria Criteria) error {
	fields := []struct {
		name  string
		value string
		valid []string
	}{
		{"ambiguity", criteria.Ambiguity, []string{"low", "medium", "high"}},
		{"risk", criteria.Risk, []string{"low", "medium", "high"}},
		{"scope", criteria.Scope, []string{"bounded", "cross_module"}},
		{"evidence", criteria.Evidence, []string{"sufficient", "insufficient", "conflicting"}},
	}
	for _, field := range fields {
		if field.value != "" && !contains(field.valid, field.value) {
			return fmt.Errorf("leveling_signal_unknown: defaults.roles.%s.criteria.%s %q is invalid", id, field.name, field.value)
		}
	}
	return nil
}

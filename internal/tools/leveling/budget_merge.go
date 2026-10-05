package leveling

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func mergeBudgetPolicy(base, override domain.EffortBudgetPolicy) domain.EffortBudgetPolicy {
	mergeBudgetLimits(&base, override)
	mergeBudgetLevels(&base, override)
	mergeBudgetPermissions(&base, override.Permissions)
	mergeBudgetEscalation(&base, override)
	return base
}

func mergeBudgetLimits(base *domain.EffortBudgetPolicy, override domain.EffortBudgetPolicy) {
	if override.MaxDelegations != 0 {
		base.MaxDelegations = override.MaxDelegations
	}
	if override.MaxDepth != 0 {
		base.MaxDepth = override.MaxDepth
	}
	if override.Parallelism != 0 {
		base.Parallelism = override.Parallelism
	}
}

func mergeBudgetLevels(base *domain.EffortBudgetPolicy, override domain.EffortBudgetPolicy) {
	if override.ContextBudget != "" {
		base.ContextBudget = override.ContextBudget
	}
	if override.EvidenceDepth != "" {
		base.EvidenceDepth = override.EvidenceDepth
	}
}

func mergeBudgetPermissions(base *domain.EffortBudgetPolicy, permissions domain.BudgetPermissions) {
	if permissions.Specialists {
		base.Permissions.Specialists = true
	}
	if permissions.SideQuests {
		base.Permissions.SideQuests = true
	}
	if permissions.DeepHistory {
		base.Permissions.DeepHistory = true
	}
}

func mergeBudgetEscalation(base *domain.EffortBudgetPolicy, override domain.EffortBudgetPolicy) {
	if override.EscalationAllowed {
		base.EscalationAllowed = true
	}
	if override.MaxEscalations != 0 {
		base.MaxEscalations = override.MaxEscalations
	}
}

type budgetOverrideDocument struct {
	Defaults struct {
		Budget *budgetOverride `yaml:"budget"`
		Roles  map[string]struct {
			Budget *budgetOverride `yaml:"budget"`
		} `yaml:"roles"`
	} `yaml:"defaults"`
}

type budgetOverride struct {
	MaxDelegations    *int                       `yaml:"max_delegations"`
	MaxDepth          *int                       `yaml:"max_depth"`
	Parallelism       *int                       `yaml:"parallelism"`
	ContextBudget     *string                    `yaml:"context_budget"`
	EvidenceDepth     *string                    `yaml:"evidence_depth"`
	Permissions       *budgetPermissionOverrides `yaml:"permissions"`
	EscalationAllowed *bool                      `yaml:"escalation_allowed"`
	MaxEscalations    *int                       `yaml:"max_escalations"`
}

type budgetPermissionOverrides struct {
	Specialists *bool `yaml:"specialists"`
	SideQuests  *bool `yaml:"side_quests"`
	DeepHistory *bool `yaml:"deep_history"`
}

func parseBudgetOverrides(raw []byte) (budgetOverrideDocument, error) {
	var document budgetOverrideDocument
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return budgetOverrideDocument{}, fmt.Errorf("decode budget fields: %w", err)
	}
	return document, nil
}

func applyExplicitBudgetOverrides(policy *Policy, document budgetOverrideDocument) {
	if document.Defaults.Budget != nil {
		policy.Defaults.Budget = applyBudgetOverride(policy.Defaults.Budget, *document.Defaults.Budget)
	}
	for id, roleOverride := range document.Defaults.Roles {
		if roleOverride.Budget == nil {
			continue
		}
		role := policy.Defaults.Roles[id]
		role.Budget = applyBudgetOverride(role.Budget, *roleOverride.Budget)
		policy.Defaults.Roles[id] = role
	}
}

func applyBudgetOverride(base domain.EffortBudgetPolicy, override budgetOverride) domain.EffortBudgetPolicy {
	applyBudgetLimits(&base, override)
	applyBudgetLevels(&base, override)
	applyBudgetPermissions(&base, override.Permissions)
	applyBudgetEscalation(&base, override)
	return base
}

func applyBudgetLimits(base *domain.EffortBudgetPolicy, override budgetOverride) {
	if override.MaxDelegations != nil {
		base.MaxDelegations = *override.MaxDelegations
	}
	if override.MaxDepth != nil {
		base.MaxDepth = *override.MaxDepth
	}
	if override.Parallelism != nil {
		base.Parallelism = *override.Parallelism
	}
}

func applyBudgetLevels(base *domain.EffortBudgetPolicy, override budgetOverride) {
	if override.ContextBudget != nil {
		base.ContextBudget = *override.ContextBudget
	}
	if override.EvidenceDepth != nil {
		base.EvidenceDepth = *override.EvidenceDepth
	}
}

func applyBudgetPermissions(base *domain.EffortBudgetPolicy, override *budgetPermissionOverrides) {
	if override == nil {
		return
	}
	if override.Specialists != nil {
		base.Permissions.Specialists = *override.Specialists
	}
	if override.SideQuests != nil {
		base.Permissions.SideQuests = *override.SideQuests
	}
	if override.DeepHistory != nil {
		base.Permissions.DeepHistory = *override.DeepHistory
	}
}

func applyBudgetEscalation(base *domain.EffortBudgetPolicy, override budgetOverride) {
	if override.EscalationAllowed != nil {
		base.EscalationAllowed = *override.EscalationAllowed
	}
	if override.MaxEscalations != nil {
		base.MaxEscalations = *override.MaxEscalations
	}
}

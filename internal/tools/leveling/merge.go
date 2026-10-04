package leveling

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Merge overlays a customer policy on the embedded defaults.
func Merge(defaults, override []byte) (Policy, error) {
	base, err := parseUnvalidated(defaults)
	if err != nil {
		return Policy{}, fmt.Errorf("leveling: parse defaults: %w", err)
	}
	if strings.TrimSpace(string(override)) == "" {
		return validateMergedPolicy(base)
	}
	custom, err := parseUnvalidated(override)
	if err != nil {
		return Policy{}, fmt.Errorf("leveling: parse override: %w", err)
	}
	if err := normalizeProviders(&custom); err != nil {
		return Policy{}, err
	}
	rankedOverrides, err := parseRankedOverrides(override)
	if err != nil {
		return Policy{}, fmt.Errorf("leveling: parse provider ranked flags: %w", err)
	}
	mergePolicy(&base, custom, rankedOverrides)
	return validateMergedPolicy(base)
}

func validateMergedPolicy(policy Policy) (Policy, error) {
	if err := normalizeProviders(&policy); err != nil {
		return Policy{}, err
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func mergePolicy(base *Policy, override Policy, rankedOverrides map[string]*bool) {
	if override.Version != 0 {
		base.Version = override.Version
	}
	mergeDefaults(&base.Defaults, override.Defaults)
	mergeRoles(&base.Defaults.Roles, override.Defaults.Roles)
	mergeProviders(&base.Providers, override.Providers, rankedOverrides)
}

func mergeDefaults(base *Defaults, override Defaults) {
	if len(override.EffortTiers) != 0 {
		base.EffortTiers = override.EffortTiers
	}
	if override.Fallback.Capability != "" {
		base.Fallback.Capability = override.Fallback.Capability
	}
	if override.Fallback.Effort != "" {
		base.Fallback.Effort = override.Fallback.Effort
	}
	if override.Fallback.Reason != "" {
		base.Fallback.Reason = override.Fallback.Reason
	}
}

func mergeRoles(base *map[string]Role, overrides map[string]Role) {
	if *base == nil {
		*base = map[string]Role{}
	}
	for id, role := range overrides {
		(*base)[id] = mergeRole((*base)[id], role)
	}
}

func mergeProviders(base *map[string]Provider, overrides map[string]Provider, rankedOverrides map[string]*bool) {
	if *base == nil {
		*base = map[string]Provider{}
	}
	for id, provider := range overrides {
		canonical := strings.ToUpper(id)
		(*base)[canonical] = mergeProvider((*base)[canonical], provider, rankedOverrides[canonical])
	}
}

func mergeRole(base, override Role) Role {
	if override.Capability != "" {
		base.Capability = override.Capability
	}
	if override.Effort != "" {
		base.Effort = override.Effort
	}
	base.Criteria = mergeCriteria(base.Criteria, override.Criteria)
	if override.Escalation.Capability != "" {
		base.Escalation.Capability = override.Escalation.Capability
	}
	if override.Escalation.Effort != "" {
		base.Escalation.Effort = override.Escalation.Effort
	}
	return base
}

func mergeCriteria(base, override Criteria) Criteria {
	if override.Ambiguity != "" {
		base.Ambiguity = override.Ambiguity
	}
	if override.Risk != "" {
		base.Risk = override.Risk
	}
	if override.Scope != "" {
		base.Scope = override.Scope
	}
	if override.Evidence != "" {
		base.Evidence = override.Evidence
	}
	return base
}

func mergeProvider(base, override Provider, rankedOverride *bool) Provider {
	if rankedOverride != nil {
		base.Ranked = *rankedOverride
	}
	if len(base.Models) == 0 {
		base.Models = map[string]string{}
	}
	for key, value := range override.Models {
		base.Models[key] = value
	}
	for key, value := range override.Display {
		if base.Display == nil {
			base.Display = map[string]string{}
		}
		base.Display[key] = value
	}
	if len(override.EffortTiers) != 0 {
		base.EffortTiers = override.EffortTiers
	}
	return base
}

func parseRankedOverrides(raw []byte) (map[string]*bool, error) {
	var document struct {
		Providers map[string]struct {
			Ranked *bool `yaml:"ranked"`
		} `yaml:"providers"`
	}
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("decode ranked flags: %w", err)
	}
	result := make(map[string]*bool, len(document.Providers))
	for id, provider := range document.Providers {
		if provider.Ranked != nil {
			result[strings.ToUpper(strings.TrimSpace(id))] = provider.Ranked
		}
	}
	return result, nil
}

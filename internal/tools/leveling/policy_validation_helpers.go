package leveling

import (
	"fmt"
	"strings"
)

func validateCapabilityMappings(defaults Defaults, providers map[string]Provider) error {
	known := declaredCapabilities(defaults)
	for providerID, provider := range providers {
		if err := validateProviderCapabilities(providerID, provider, known); err != nil {
			return err
		}
	}
	return nil
}

func declaredCapabilities(defaults Defaults) map[string]bool {
	known := map[string]bool{
		"general":                    true,
		"economical":                 true,
		"reasoning":                  true,
		defaults.Fallback.Capability: true,
	}
	for _, role := range defaults.Roles {
		known[role.Capability] = true
		if role.Escalation.Capability != "" {
			known[role.Escalation.Capability] = true
		}
	}
	return known
}

func validateProviderCapabilities(providerID string, provider Provider, known map[string]bool) error {
	for capability := range provider.Models {
		if !known[capability] {
			return fmt.Errorf("leveling_mapping_invalid: providers.%s.models.%s references undeclared capability", providerID, capability)
		}
	}
	return nil
}

func validateCapability(path, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("leveling_mapping_invalid: %s must not be empty", path)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("leveling_mapping_invalid: %s must not contain surrounding whitespace", path)
	}
	return nil
}

func providerPath(id string) string {
	return "providers." + id
}

func validateEffortTiers(path string, tiers []string) error {
	if len(tiers) == 0 {
		return fmt.Errorf("leveling_mapping_invalid: %s must not be empty", path)
	}
	seen := make(map[string]bool, len(tiers))
	for _, tier := range tiers {
		if !effortTiers[tier] {
			return fmt.Errorf("leveling_mapping_invalid: %s contains unknown effort tier %q", path, tier)
		}
		if seen[tier] {
			return fmt.Errorf("leveling_mapping_invalid: %s contains duplicate effort tier %q", path, tier)
		}
		seen[tier] = true
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

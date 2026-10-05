package leveling

import (
	"fmt"
	"strings"
)

func validateProvider(id string, provider Provider) error {
	if strings.TrimSpace(id) == "" || len(provider.Models) == 0 {
		return fmt.Errorf("leveling_mapping_invalid: providers.%s requires models", id)
	}
	if err := validateProviderModels(id, provider.Models); err != nil {
		return err
	}
	if err := validateProviderDisplay(id, provider.Display); err != nil {
		return err
	}
	return validateEffortTiers("providers."+id+".effort_tiers", provider.EffortTiers)
}

func validateProviderModels(id string, models map[string]string) error {
	for capability, model := range models {
		if invalidProviderMapping(capability, model) {
			return fmt.Errorf("leveling_mapping_invalid: %s mapping requires non-empty capability and model", providerPath(id))
		}
	}
	return nil
}

func invalidProviderMapping(capability, model string) bool {
	return capability != strings.TrimSpace(capability) || model != strings.TrimSpace(model) || strings.TrimSpace(capability) == "" || strings.TrimSpace(model) == ""
}

func validateProviderDisplay(id string, display map[string]string) error {
	for model, name := range display {
		if invalidProviderDisplay(model, name) {
			return fmt.Errorf("leveling_mapping_invalid: %s display requires a non-empty single-line name for each model", providerPath(id))
		}
	}
	return nil
}

func invalidProviderDisplay(model, name string) bool {
	return strings.TrimSpace(model) == "" || strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\r\n")
}

package config

import (
	"fmt"
	"slices"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Parity compares persisted bindings with the configuration and returns one
// finding per drift: a provider that is not declared or enabled, a pinned model
// that differs, or a capability the operator did not allow. It exists because a
// second persistence target without a parity check is how active.yaml and
// plugins.lock once disagreed.
func Parity(file File, bindings []integration.Binding) []string {
	var findings []string
	for _, binding := range bindings {
		if finding := parityFinding(file, binding); finding != "" {
			findings = append(findings, finding)
		}
	}
	return findings
}

func parityFinding(file File, binding integration.Binding) string {
	provider, declared := file.Providers[binding.Provider]
	switch {
	case !declared:
		return fmt.Sprintf("binding %s/%s: provider is not declared", binding.Provider, binding.Consumer)
	case !provider.Enabled:
		return fmt.Sprintf("binding %s/%s: provider is disabled", binding.Provider, binding.Consumer)
	case binding.Model != provider.Model && !provider.AllowModelAlias:
		return fmt.Sprintf("binding %s/%s: model differs from the pinned model", binding.Provider, binding.Consumer)
	case !slices.Contains(provider.AllowedCapabilities, binding.Capability):
		return fmt.Sprintf("binding %s/%s: capability %s is not allowed", binding.Provider, binding.Consumer, binding.Capability)
	}
	return ""
}

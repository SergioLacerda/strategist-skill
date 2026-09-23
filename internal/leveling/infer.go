package leveling

import (
	"sort"
	"strings"
)

// ProviderForModel returns the ranked policy provider a host model id belongs
// to: the provider whose model table lists the id, or whose lower-cased key
// prefixes it (`claude-opus-5` → CLAUDE). Empty when no ranked provider
// matches, so an unknown vendor never borrows another provider's mapping.
func (p Policy) ProviderForModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return ""
	}
	keys := make([]string, 0, len(p.Providers))
	for key := range p.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if providerOwnsModel(key, p.Providers[key], model) {
			return key
		}
	}
	return ""
}

func providerOwnsModel(key string, provider Provider, model string) bool {
	if !provider.Ranked {
		return false
	}
	for _, id := range provider.Models {
		if strings.EqualFold(id, model) {
			return true
		}
	}
	return strings.HasPrefix(model, strings.ToLower(key)+"-")
}

// ResolveLevelInferred resolves a level in automatic mode when no provider was
// given: the provider is inferred from the host-reported model, so the role
// on_start hook (which passes --host-model but no --provider) still gets the
// policy's effort. The policy is loaded only when the host report is
// incomplete; without a host model nothing can be inferred and the level is
// resolved from the host values alone. Manual mode must not call this: it is
// host passthrough and never reads the policy.
func ResolveLevelInferred(load PolicyLoader, role string, signals Signals, host Host) (Level, error) {
	hostOnly, err := ResolveLevelLazy(load, "", role, signals, host)
	if err != nil || strings.TrimSpace(host.Model) == "" || (hostOnly.Model != "" && hostOnly.Effort != "") {
		return hostOnly, err
	}
	policy, err := load()
	if err != nil {
		return Level{}, err
	}
	provider := policy.ProviderForModel(host.Model)
	if provider == "" {
		return hostOnly, nil
	}
	return ResolveLevelLazy(func() (Policy, error) { return policy, nil }, provider, role, signals, host)
}

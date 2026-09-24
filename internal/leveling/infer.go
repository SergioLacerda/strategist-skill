package leveling

import (
	"sort"
	"strings"
)

// ProviderMatch describes how a model was associated with a provider.
type ProviderMatch string

// Provider match kinds, from strongest to weakest association.
const (
	ProviderMatchExact    ProviderMatch = "exact"
	ProviderMatchInferred ProviderMatch = "prefix_inferred"
	ProviderMatchUnknown  ProviderMatch = "unknown"
)

// ProviderResolution preserves the quality of a provider identity match.
type ProviderResolution struct {
	Provider string
	Match    ProviderMatch
}

// ProviderForModel returns the ranked policy provider a host model id belongs
// to: the provider whose model table lists the id, or whose lower-cased key
// prefixes it (`claude-opus-5` → CLAUDE). Empty when no ranked provider
// matches, so an unknown vendor never borrows another provider's mapping.
func (p Policy) ProviderForModel(model string) string {
	return p.MatchProvider(model).Provider
}

// MatchProvider resolves exact identities before compatibility prefixes so an
// inferred prefix can never shadow a configured exact model mapping.
func (p Policy) MatchProvider(model string) ProviderResolution {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return ProviderResolution{Match: ProviderMatchUnknown}
	}
	keys := p.sortedProviderKeys()
	if key, ok := firstProvider(keys, func(key string) bool { return providerOwnsExactModel(p.Providers[key], model) }); ok {
		return ProviderResolution{Provider: key, Match: ProviderMatchExact}
	}
	if key, ok := firstProvider(keys, func(key string) bool { return providerOwnsPrefix(key, p.Providers[key], model) }); ok {
		return ProviderResolution{Provider: key, Match: ProviderMatchInferred}
	}
	return ProviderResolution{Match: ProviderMatchUnknown}
}

func (p Policy) sortedProviderKeys() []string {
	keys := make([]string, 0, len(p.Providers))
	for key := range p.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// firstProvider returns the first key, in order, that satisfies owns.
func firstProvider(keys []string, owns func(key string) bool) (string, bool) {
	for _, key := range keys {
		if owns(key) {
			return key, true
		}
	}
	return "", false
}

func providerOwnsExactModel(provider Provider, model string) bool {
	if !provider.Ranked {
		return false
	}
	for _, id := range provider.Models {
		if strings.EqualFold(id, model) {
			return true
		}
	}
	return false
}

func providerOwnsPrefix(key string, provider Provider, model string) bool {
	return provider.Ranked && strings.HasPrefix(model, strings.ToLower(key)+"-")
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
	resolution := policy.MatchProvider(host.Model)
	if resolution.Provider == "" || resolution.Match == ProviderMatchInferred {
		hostOnly.ProviderMatch = string(resolution.Match)
		return hostOnly, nil
	}
	level, err := ResolveLevelLazy(func() (Policy, error) { return policy, nil }, resolution.Provider, role, signals, host)
	if err == nil {
		level.ProviderMatch = string(resolution.Match)
	}
	return level, err
}

package leveling

import (
	"fmt"
	"strings"
)

// Level sources recorded next to every resolved level, in precedence order.
const (
	SourceHost   = "host"
	SourcePolicy = "policy"
)

// Host carries the model and effort the host reports for the running agent.
type Host struct {
	Model  string
	Effort string
}

// Level is the model and effort a role runs at, plus where the value came from.
type Level struct {
	Role           string `json:"role" yaml:"role"`
	Model          string `json:"model" yaml:"model"`
	Effort         string `json:"effort" yaml:"effort"`
	Source         string `json:"level_source" yaml:"level_source"`
	Provider       string `json:"provider,omitempty" yaml:"provider,omitempty"`
	ProviderMatch  string `json:"provider_match,omitempty" yaml:"provider_match,omitempty"`
	ModelSource    string `json:"model_source,omitempty" yaml:"model_source,omitempty"`
	EffortSource   string `json:"effort_source,omitempty" yaml:"effort_source,omitempty"`
	Capability     string `json:"capability,omitempty" yaml:"capability,omitempty"`
	FallbackUsed   bool   `json:"fallback_used,omitempty" yaml:"fallback_used,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty" yaml:"fallback_reason,omitempty"`
	PolicyVersion  int    `json:"policy_version,omitempty" yaml:"policy_version,omitempty"`
	PolicyDigest   string `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
}

// PolicyLoader loads the LEVELING policy. It is called at most once, and only
// when a value is still missing after the cheaper sources.
type PolicyLoader func() (Policy, error)

// ResolveLevel resolves the level of a role from an already loaded policy.
// Host-reported values win over the policy suggestion; a partial host report is
// completed from the policy when a provider is given. Without a provider or host
// data the level is unknown.
func ResolveLevel(policy Policy, provider, role string, signals Signals, host Host) (Level, error) {
	return ResolveLevelLazy(func() (Policy, error) { return policy, nil }, provider, role, signals, host)
}

// ResolveLevelLazy resolves the level of a role field by field with the
// precedence host, then policy. Level.Source names the highest-precedence
// source that contributed. An explicit provider is authority-bearing: even a
// complete host report loads policy so provider, capability, version, digest,
// display mapping, and provider-specific effort validation are preserved. An
// empty provider remains host passthrough and never reads policy.
func ResolveLevelLazy(load PolicyLoader, provider, role string, signals Signals, host Host) (Level, error) {
	level := Level{Role: NormalizeRole(role)}
	fillLevelSource(&level, SourceHost, host.Model, host.Effort)
	if strings.TrimSpace(provider) == "" {
		// No policy is loaded on this path, so the model id can only be
		// shortened by the policy-free fallback. This is intentional for
		// manual/host-passthrough mode: no provider authority was supplied.
		level.Model = displayFallback(provider, level.Model)
		return level, nil
	}
	return completeLevelFromPolicy(load, level, provider, role, signals)
}

// fillLevelSource records a source's model and effort. The model is kept as the
// raw id so a later policy load can still match it against the configured
// display names; shortening happens once, at the end of resolution.
func fillLevelSource(level *Level, source, model, effort string) {
	model, effort = strings.TrimSpace(model), strings.ToLower(strings.TrimSpace(effort))
	setPrimarySource(level, source, model, effort)
	fillModel(level, source, model)
	fillEffort(level, source, effort)
}

func setPrimarySource(level *Level, source, model, effort string) {
	if level.Source == "" && ((level.Model == "" && model != "") || (level.Effort == "" && effort != "")) {
		level.Source = source
	}
}

func fillModel(level *Level, source, model string) {
	if level.Model == "" && model != "" {
		level.Model, level.ModelSource = model, source
	}
}

func fillEffort(level *Level, source, effort string) {
	if level.Effort == "" && effort != "" {
		level.Effort, level.EffortSource = effort, source
	}
}

func completeLevelFromPolicy(load PolicyLoader, level Level, provider, role string, signals Signals) (Level, error) {
	policy, err := load()
	if err != nil {
		return Level{}, err
	}
	if level.Effort != "" {
		providerID := strings.ToUpper(strings.TrimSpace(provider))
		switch policy.ProviderSupportsEffort(provider, level.Effort) {
		case EffortEligible:
			// Continue below; the host-reported effort is valid for this provider.
		case EffortProviderUnknown:
			return Level{}, fmt.Errorf("%s: provider %q is not declared in the LEVELING policy", ReasonLevelingProviderUnknown, providerID)
		case EffortProviderNotRanked:
			return Level{}, fmt.Errorf("%s: provider %q is not a ranked provider in the LEVELING policy", ReasonLevelingProviderNotRanked, providerID)
		case EffortTierUnsupported:
			return Level{}, fmt.Errorf("%s: provider %q cannot execute host effort %q for role %q", ReasonLevelingRankedProviderIneligible, providerID, level.Effort, strings.ToLower(strings.TrimSpace(role)))
		}
	}
	suggestion, err := Suggest(policy, provider, role, signals)
	if err != nil {
		return Level{}, err
	}
	applyPolicyProvenance(&level, suggestion)
	// A model already answered by the host is shortened through the same
	// display table as a policy-selected one, so the label reads `Sonnet`
	// rather than `Claude-sonnet-5` whichever source supplied it.
	level.Model = policy.DisplayName(suggestion.Provider, level.Model)
	fillModel(&level, SourcePolicy, policy.DisplayName(suggestion.Provider, suggestion.Model))
	fillEffort(&level, SourcePolicy, suggestion.Effort)
	return level, nil
}

func applyPolicyProvenance(level *Level, suggestion Suggestion) {
	if level.Source == "" {
		level.Source = SourcePolicy
	}
	level.Provider, level.Capability = suggestion.Provider, suggestion.Capability
	level.FallbackUsed, level.FallbackReason = suggestion.FallbackUsed, suggestion.FallbackReason
	level.PolicyVersion, level.PolicyDigest = suggestion.PolicyVersion, suggestion.PolicyDigest
}

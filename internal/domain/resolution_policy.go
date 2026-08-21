package domain

import "fmt"

// ResolutionPolicy controls agent behavior when a configured slot provider passes
// static `strategist check` validation (valid skill.yaml, matching risk_score) but
// turns out not to be invocable at mission time, and a compatible native role
// exists for the same slot. See docs/adr/0028-native-role-resilient-baseline.md.
type ResolutionPolicy string

const (
	// ResolutionPolicyBlock preserves strict failure behavior: role_invocation_failed
	// stops the mission, exactly as before ADR-0028. No native fallback is offered.
	ResolutionPolicyBlock ResolutionPolicy = "block"
	// ResolutionPolicyAsk requests explicit user confirmation before using the
	// compatible native role for this mission. This is the recommended default.
	ResolutionPolicyAsk ResolutionPolicy = "ask"
	// ResolutionPolicyNative uses the compatible native role automatically, while
	// requiring the agent to emit degradation evidence (configured provider,
	// effective provider, reason). Never implies Approval Gate acceptance.
	ResolutionPolicyNative ResolutionPolicy = "native"
)

// DefaultResolutionPolicy is applied when active.yaml omits provider_resolution_policy
// or sets it to the empty string. ADR-0028 recommends "ask" as the default.
const DefaultResolutionPolicy = ResolutionPolicyAsk

var validResolutionPolicies = map[ResolutionPolicy]bool{
	ResolutionPolicyBlock:  true,
	ResolutionPolicyAsk:    true,
	ResolutionPolicyNative: true,
}

// Validate returns an error if the policy is set to an unrecognized value. An
// empty policy is valid — EffectivePolicy resolves it to DefaultResolutionPolicy.
func (p ResolutionPolicy) Validate() error {
	if p == "" {
		return nil
	}
	if !validResolutionPolicies[p] {
		return fmt.Errorf("provider_resolution_policy %q is not one of block, ask, native", p)
	}
	return nil
}

// EffectivePolicy returns p, or DefaultResolutionPolicy when p is empty.
func (p ResolutionPolicy) EffectivePolicy() ResolutionPolicy {
	if p == "" {
		return DefaultResolutionPolicy
	}
	return p
}

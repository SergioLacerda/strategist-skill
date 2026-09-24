package domain

import "fmt"

// ResolutionPolicy is a historical record type for retired provider-fallback
// evidence. Active mission routing must not consume it.
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

// DefaultResolutionPolicy is retained for historical outcome reconstruction.
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

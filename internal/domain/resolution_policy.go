package domain

import "fmt"

// ResolutionPolicy is the retired provider_resolution_policy value. It survives
// only so a stale active.yaml that still declares it is recognized and rejected
// (ActiveConfig.ProviderResolutionPolicy); active mission routing never consumes
// it, and there is no native fallback to choose between.
type ResolutionPolicy string

const (
	// ResolutionPolicyBlock is a value the retired policy once accepted.
	ResolutionPolicyBlock ResolutionPolicy = "block"
	// ResolutionPolicyAsk is a value the retired policy once accepted.
	ResolutionPolicyAsk ResolutionPolicy = "ask"
	// ResolutionPolicyNative is a value the retired policy once accepted.
	ResolutionPolicyNative ResolutionPolicy = "native"
)

var validResolutionPolicies = map[ResolutionPolicy]bool{
	ResolutionPolicyBlock:  true,
	ResolutionPolicyAsk:    true,
	ResolutionPolicyNative: true,
}

// Validate returns an error if the policy is set to an unrecognized value. An
// empty policy is valid.
func (p ResolutionPolicy) Validate() error {
	if p == "" {
		return nil
	}
	if !validResolutionPolicies[p] {
		return fmt.Errorf("provider_resolution_policy %q is not one of block, ask, native", p)
	}
	return nil
}

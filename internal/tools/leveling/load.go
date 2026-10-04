package leveling

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// EffectivePolicy contains the validated policy selected for a runtime
// boundary and the identities needed to diagnose source/runtime drift.
type EffectivePolicy struct {
	Policy         Policy
	Source         string
	DefaultVersion int
	DefaultDigest  string
	OverrideDigest string
}

// LoadEffective merges a customer document over the authoritative embedded
// defaults. An empty override means that the embedded policy is selected.
// The returned identities distinguish an intentional customer override from a
// stale embedded source without requiring a repository checkout.
func LoadEffective(defaults, override []byte, source string) (EffectivePolicy, error) {
	defaultPolicy, err := Parse(defaults)
	if err != nil {
		return EffectivePolicy{}, fmt.Errorf("leveling: parse authoritative defaults: %w", err)
	}
	policy, err := Merge(defaults, override)
	if err != nil {
		return EffectivePolicy{}, err
	}
	result := EffectivePolicy{
		Policy:         policy,
		Source:         source,
		DefaultVersion: defaultPolicy.Version,
		DefaultDigest:  defaultPolicy.Digest(),
	}
	if strings.TrimSpace(string(override)) != "" {
		overridePolicy, parseErr := parseUnvalidated(override)
		if parseErr != nil {
			return EffectivePolicy{}, fmt.Errorf("leveling: parse override identity: %w", parseErr)
		}
		result.OverrideDigest = overridePolicy.Digest()
	}
	return result, nil
}

// VerifyDefaultAuthority confirms that an install manifest was produced from
// the same embedded LEVELING source as the running executable.
func VerifyDefaultAuthority(defaults []byte, expectedVersion int, expectedDigest string) error {
	policy, err := Parse(defaults)
	if err != nil {
		return fmt.Errorf("leveling_policy_stale: parse authoritative defaults: %w", err)
	}
	if expectedVersion != policy.Version || strings.TrimSpace(expectedDigest) != policy.Digest() {
		return fmt.Errorf("leveling_policy_stale: expected version=%d digest=%s, observed version=%d digest=%s", expectedVersion, expectedDigest, policy.Version, policy.Digest())
	}
	return nil
}

// Parse decodes and validates a customer or embedded LEVELING YAML document.
func Parse(raw []byte) (Policy, error) {
	var policy Policy
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("leveling: parse policy: %w", err)
	}
	if err := normalizeProviders(&policy); err != nil {
		return Policy{}, err
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

// LoadFile reads, decodes, and validates a LEVELING policy from disk.
func LoadFile(path string) (Policy, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller selects the customer policy path
	if err != nil {
		return Policy{}, fmt.Errorf("leveling: read %s: %w", path, err)
	}
	return Parse(raw)
}

// CheckDefaultParity verifies that two default policy documents resolve to
// the same normalized policy. It is intended for source/runtime CI checks;
// customer overrides are intentionally not required to remain identical.
func CheckDefaultParity(source, runtime []byte) error {
	sourcePolicy, err := Parse(source)
	if err != nil {
		return fmt.Errorf("leveling: parse source default: %w", err)
	}
	runtimePolicy, err := Parse(runtime)
	if err != nil {
		return fmt.Errorf("leveling: parse runtime default: %w", err)
	}
	if sourcePolicy.Digest() != runtimePolicy.Digest() {
		return fmt.Errorf("leveling: source/runtime default digest mismatch: source=%s runtime=%s", sourcePolicy.Digest(), runtimePolicy.Digest())
	}
	return nil
}

func parseUnvalidated(raw []byte) (Policy, error) {
	var policy Policy
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("leveling: decode policy: %w", err)
	}
	return policy, nil
}

func normalizeProviders(policy *Policy) error {
	if len(policy.Providers) == 0 {
		return nil
	}
	normalized := make(map[string]Provider, len(policy.Providers))
	for id, provider := range policy.Providers {
		canonical := strings.ToUpper(strings.TrimSpace(id))
		if canonical == "" {
			return fmt.Errorf("leveling: provider identifier must not be empty")
		}
		if _, exists := normalized[canonical]; exists {
			return fmt.Errorf("leveling: duplicate provider identifier %q", canonical)
		}
		normalized[canonical] = provider
	}
	policy.Providers = normalized
	return nil
}

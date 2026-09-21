// Package leveling selects a model and effort tier for a Strategist role.
// LEVELING is provider-neutral; provider profiles only translate its tiers.
package leveling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// SkillName is the user-facing name of the model/effort selection skill.
const SkillName = "LEVELING"

var effortTiers = map[string]bool{
	"none": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true,
}

// Policy is the complete provider-neutral LEVELING configuration.
type Policy struct {
	Version   int                 `yaml:"version"`
	Defaults  Defaults            `yaml:"defaults"`
	Providers map[string]Provider `yaml:"providers"`
}

// Defaults contains generic role, effort, and provider-fallback policy.
type Defaults struct {
	EffortTiers []string        `yaml:"effort_tiers"`
	Fallback    Fallback        `yaml:"fallback"`
	Roles       map[string]Role `yaml:"roles"`
}

// Role describes the normal and escalated recommendation for a Strategist role.
type Role struct {
	Capability string     `yaml:"capability"`
	Effort     string     `yaml:"effort"`
	Criteria   Criteria   `yaml:"criteria"`
	Escalation Escalation `yaml:"escalation"`
}

// Criteria records the customer-editable signals used to assess a role.
type Criteria struct {
	Ambiguity string `yaml:"ambiguity"`
	Risk      string `yaml:"risk"`
	Scope     string `yaml:"scope"`
	Evidence  string `yaml:"evidence"`
}

// Escalation is the stronger recommendation used for elevated signals.
type Escalation struct {
	Capability string `yaml:"capability"`
	Effort     string `yaml:"effort"`
}

// Fallback describes the provider-neutral recommendation for unknown providers.
type Fallback struct {
	Capability string `yaml:"capability"`
	Effort     string `yaml:"effort"`
	Reason     string `yaml:"reason"`
}

// Provider translates capabilities and effort tiers for one ranked provider.
type Provider struct {
	Ranked      bool              `yaml:"ranked"`
	Models      map[string]string `yaml:"models"`
	EffortTiers []string          `yaml:"effort_tiers"`
	// Display maps a model id to the short name shown on role log lines. It is
	// presentation only, so it is excluded from the policy digest (json:"-").
	Display map[string]string `yaml:"display,omitempty" json:"-"`
}

// Signals are runtime facts used to decide whether a role should escalate.
type Signals struct {
	Ambiguity           string
	Risk                string
	Scope               string
	Evidence            string
	ArchitecturalChange bool
	SecuritySensitive   bool
	ConflictingEvidence bool
	RepeatedFailures    int
}

// Suggestion is the resolved model, capability, and effort recommendation.
type Suggestion struct {
	Provider       string   `json:"provider" yaml:"provider"`
	Role           string   `json:"role" yaml:"role"`
	Model          string   `json:"model" yaml:"model"`
	Capability     string   `json:"capability" yaml:"capability"`
	Effort         string   `json:"effort" yaml:"effort"`
	Rationale      []string `json:"rationale" yaml:"rationale"`
	FallbackUsed   bool     `json:"fallback_used" yaml:"fallback_used"`
	FallbackReason string   `json:"fallback_reason,omitempty" yaml:"fallback_reason,omitempty"`
	PolicyVersion  int      `json:"policy_version" yaml:"policy_version"`
	PolicyDigest   string   `json:"policy_digest" yaml:"policy_digest"`
}

// Digest returns the stable SHA-256 identity of the normalized policy.
func (p Policy) Digest() string {
	encoded, err := json.Marshal(p) // encoding/json orders map keys deterministically.
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// VerifyDigest rejects a stale policy when its normalized identity differs
// from the digest recorded by the installer or activation caller.
func (p Policy) VerifyDigest(expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return fmt.Errorf("leveling_policy_digest_mismatch: expected policy digest must not be empty")
	}
	actual := p.Digest()
	if actual != expected {
		return fmt.Errorf("leveling_policy_digest_mismatch: policy digest mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

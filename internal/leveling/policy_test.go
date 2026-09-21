package leveling_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultPolicy(t *testing.T) leveling.Policy {
	t.Helper()
	raw, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(raw)
	require.NoError(t, err)
	return policy
}

func TestDefaultPolicyHasOnlyCodexAndClaude(t *testing.T) {
	policy := defaultPolicy(t)
	assert.Len(t, policy.Providers, 2)
	assert.Contains(t, policy.Providers, "CODEX")
	assert.Contains(t, policy.Providers, "CLAUDE")
}

func TestSuggestUsesProviderMapping(t *testing.T) {
	policy := defaultPolicy(t)
	got, err := leveling.Suggest(policy, "codex", "scout", leveling.Signals{})
	require.NoError(t, err)
	assert.Equal(t, "CODEX", got.Provider)
	assert.Equal(t, "codex-economical", got.Model)
	assert.False(t, got.FallbackUsed)
	assert.NotEmpty(t, got.PolicyDigest)
}

func TestSuggestEscalatesHighUncertainty(t *testing.T) {
	policy := defaultPolicy(t)
	got, err := leveling.Suggest(policy, "CLAUDE", "scout", leveling.Signals{SecuritySensitive: true})
	require.NoError(t, err)
	assert.Equal(t, "reasoning", got.Capability)
	assert.Equal(t, "high", got.Effort)
	assert.Contains(t, got.Rationale, "security_sensitive")
}

func TestSuggestUsesCustomerCriteriaThresholds(t *testing.T) {
	policy := defaultPolicy(t)
	role := policy.Defaults.Roles["scout"]
	role.Criteria.Ambiguity = "high"
	policy.Defaults.Roles["scout"] = role
	got, err := leveling.Suggest(policy, "CODEX", "scout", leveling.Signals{Ambiguity: "medium"})
	require.NoError(t, err)
	assert.Equal(t, "economical", got.Capability)
	role.Criteria.Ambiguity = "low"
	policy.Defaults.Roles["scout"] = role
	got, err = leveling.Suggest(policy, "CODEX", "scout", leveling.Signals{Ambiguity: "medium"})
	require.NoError(t, err)
	assert.Equal(t, "reasoning", got.Capability)
}

func TestSuggestUnknownProviderUsesGenericFallback(t *testing.T) {
	policy := defaultPolicy(t)
	got, err := leveling.Suggest(policy, "new-ranked", "ranger", leveling.Signals{})
	require.NoError(t, err)
	assert.True(t, got.FallbackUsed)
	assert.Equal(t, "generic_ranked_provider", got.FallbackReason)
	assert.Empty(t, got.Model)
	assert.NotContains(t, got.Rationale, "codex")
}

func TestSuggestUsesProviderAddedByCustomerYAML(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Merge(defaults, []byte("version: 1\nproviders:\n  ACME:\n    ranked: true\n    models: {economical: acme-small, reasoning: acme-large}\n    effort_tiers: [none, low, medium, high, xhigh, max]\n"))
	require.NoError(t, err)
	got, err := leveling.Suggest(policy, "acme", "scout", leveling.Signals{})
	require.NoError(t, err)
	assert.Equal(t, "acme-small", got.Model)
	assert.False(t, got.FallbackUsed)
}

func TestSuggestRejectsUnsupportedEffort(t *testing.T) {
	policy := defaultPolicy(t)
	policy.Defaults.Roles["scout"] = leveling.Role{Capability: "reasoning", Effort: "max"}
	_, err := leveling.Suggest(policy, "CLAUDE", "scout", leveling.Signals{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot execute effort")
}

func TestMergePreservesUntouchedDefaults(t *testing.T) {
	raw, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Merge(raw, []byte("version: 1\ndefaults:\n  roles:\n    scout:\n      effort: low\n"))
	require.NoError(t, err)
	assert.Equal(t, "low", policy.Defaults.Roles["scout"].Effort)
	assert.Equal(t, "economical", policy.Defaults.Roles["scout"].Capability)
	assert.Contains(t, policy.Providers, "CODEX")
}

func TestParseRejectsUnknownFieldAndTier(t *testing.T) {
	raw := []byte("version: 1\ndefaults:\n  effort_tiers: [fast]\n  fallback: {capability: general, effort: fast, reason: test}\nproviders: {CODEX: {models: {general: x}, effort_tiers: [fast]}}\n")
	_, err := leveling.Parse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown effort tier")
}

func TestParseRejectsMalformedYAML(t *testing.T) {
	_, err := leveling.Parse([]byte("version: ["))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse policy")
}

func TestValidateRejectsIncompletePolicies(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*leveling.Policy)
		want   string
	}{
		{"version", func(p *leveling.Policy) { p.Version = 0 }, "version must be positive"},
		{"effort tiers", func(p *leveling.Policy) { p.Defaults.EffortTiers = nil }, "must not be empty"},
		{"fallback", func(p *leveling.Policy) { p.Defaults.Fallback.Reason = "" }, "fallback requires"},
		{"role", func(p *leveling.Policy) { p.Defaults.Roles["scout"] = leveling.Role{} }, "leveling_mapping_invalid"},
		{"provider", func(p *leveling.Policy) { p.Providers["CODEX"] = leveling.Provider{} }, "requires models"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := defaultPolicy(t)
			policy := base
			tt.mutate(&policy)
			err := policy.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestLoadFileReadsCustomerPolicy(t *testing.T) {
	raw, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "leveling.yaml")
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	_, err = leveling.LoadFile(path)
	require.NoError(t, err)
}

func TestLoadFileReportsMissingFile(t *testing.T) {
	_, err := leveling.LoadFile(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestMergeOverridesEveryPolicyLayer(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	override := []byte(`version: 2
defaults:
  effort_tiers: [none, low, medium, high, xhigh, max]
  fallback: {capability: custom, effort: low, reason: custom_reason}
  roles:
    scout:
      capability: custom_capability
      effort: low
      criteria: {ambiguity: high, risk: high, scope: cross_module, evidence: insufficient}
      escalation: {capability: custom_escalation, effort: max}
providers:
  codex:
    ranked: true
    models: {custom_capability: custom-model}
    effort_tiers: [low]
`)
	policy, err := leveling.Merge(defaults, override)
	require.NoError(t, err)
	assert.Equal(t, 2, policy.Version)
	assert.Equal(t, "custom", policy.Defaults.Fallback.Capability)
	assert.Equal(t, "custom_capability", policy.Defaults.Roles["scout"].Capability)
	assert.Equal(t, "high", policy.Defaults.Roles["scout"].Criteria.Risk)
	assert.Equal(t, "custom_escalation", policy.Defaults.Roles["scout"].Escalation.Capability)
	assert.Equal(t, "custom-model", policy.Providers["CODEX"].Models["custom_capability"])
}

func TestMergeRejectsDuplicateProviderIdentifiers(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	_, err = leveling.Merge(defaults, []byte("version: 1\nproviders:\n  codex: {models: {general: x}, effort_tiers: [medium]}\n  CODEX: {models: {general: y}, effort_tiers: [medium]}\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate provider")
}

func TestMergeProviderAndCriteriaOverrides(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	override := []byte("version: 1\ndefaults:\n  roles:\n    scout:\n      criteria: {ambiguity: high}\nproviders:\n  CLAUDE:\n    ranked: false\n    models: {reasoning: claude-custom}\n    effort_tiers: [medium, high]\n")
	policy, err := leveling.Merge(defaults, override)
	require.NoError(t, err)
	assert.Equal(t, "high", policy.Defaults.Roles["scout"].Criteria.Ambiguity)
	assert.Equal(t, "claude-custom", policy.Providers["CLAUDE"].Models["reasoning"])
	assert.False(t, policy.Providers["CLAUDE"].Ranked)
	assert.Equal(t, []string{"medium", "high"}, policy.Providers["CLAUDE"].EffortTiers)
}

func TestMergeHandlesEmptyAndInvalidInputs(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	merged, err := leveling.Merge(defaults, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultPolicy(t).Digest(), merged.Digest())
	_, err = leveling.Merge([]byte("version: ["), nil)
	require.Error(t, err)
	_, err = leveling.Merge(defaults, []byte("version: ["))
	require.Error(t, err)
}

func TestValidateRejectsDuplicateAndInvalidEscalationTiers(t *testing.T) {
	policy := defaultPolicy(t)
	policy.Defaults.EffortTiers = []string{"medium", "medium"}
	require.ErrorContains(t, policy.Validate(), "duplicate effort tier")
	policy = defaultPolicy(t)
	policy.Defaults.Roles["scout"] = leveling.Role{
		Capability: "economical", Effort: "medium",
		Escalation: leveling.Escalation{Capability: "reasoning", Effort: "bogus"},
	}
	require.ErrorContains(t, policy.Validate(), "escalation.effort")
	policy = defaultPolicy(t)
	policy.Defaults.Roles["scout"] = leveling.Role{Capability: "economical", Effort: "medium", Criteria: leveling.Criteria{Risk: "extreme"}}
	require.ErrorContains(t, policy.Validate(), "criteria.risk")
}

func TestSuggestRejectsMissingCapabilityMapping(t *testing.T) {
	policy := defaultPolicy(t)
	policy.Defaults.Roles["scout"] = leveling.Role{Capability: "missing", Effort: "medium"}
	_, err := leveling.Suggest(policy, "CODEX", "scout", leveling.Signals{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no model mapping")
}

func TestSuggestUsesGenericRoleFallback(t *testing.T) {
	policy := defaultPolicy(t)
	got, err := leveling.Suggest(policy, "UNKNOWN", "unlisted", leveling.Signals{})
	require.NoError(t, err)
	assert.Equal(t, policy.Defaults.Fallback.Capability, got.Capability)
	assert.Equal(t, policy.Defaults.Fallback.Effort, got.Effort)
	assert.True(t, got.FallbackUsed)
}

func TestSuggestCoversEscalationReasons(t *testing.T) {
	policy := defaultPolicy(t)
	tests := []struct {
		name    string
		signals leveling.Signals
		want    string
	}{
		{"architecture", leveling.Signals{ArchitecturalChange: true}, "architectural_change"},
		{"conflict", leveling.Signals{ConflictingEvidence: true}, "conflicting_evidence"},
		{"failure", leveling.Signals{RepeatedFailures: 2}, "repeated_failure"},
		{"ambiguity", leveling.Signals{Ambiguity: "high"}, "high_ambiguity"},
		{"risk", leveling.Signals{Risk: "high"}, "high_risk"},
		{"scope", leveling.Signals{Scope: "cross_module"}, "cross_module_scope"},
		{"evidence", leveling.Signals{Evidence: "insufficient"}, "insufficient_evidence"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := leveling.Suggest(policy, "CODEX", "scout", tt.signals)
			require.NoError(t, err)
			assert.Contains(t, got.Rationale, tt.want)
		})
	}
}

func TestDefaultPolicySourceAndRuntimeHaveSameDigest(t *testing.T) {
	source, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	runtime, err := os.ReadFile(filepath.Join("..", "..", ".strategist", "leveling.yaml"))
	require.NoError(t, err)
	require.NoError(t, leveling.CheckDefaultParity(source, runtime))
}

func TestVerifyDigestRejectsStalePolicy(t *testing.T) {
	policy := defaultPolicy(t)
	require.NoError(t, policy.VerifyDigest(policy.Digest()))
	err := policy.VerifyDigest("stale")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "policy digest mismatch")
}

func TestCheckDefaultParityRejectsDrift(t *testing.T) {
	source, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	drifted := bytes.Replace(source, []byte("version: 1"), []byte("version: 2"), 1)
	require.ErrorContains(t, leveling.CheckDefaultParity(source, drifted), "digest mismatch")
}

func TestLoadEffectiveMergesPartialOverrideAndExposesAuthority(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	effective, err := leveling.LoadEffective(defaults, []byte("version: 1\ndefaults:\n  roles:\n    scout:\n      effort: low\n"), ".strategist/leveling.yaml")
	require.NoError(t, err)
	assert.Equal(t, "low", effective.Policy.Defaults.Roles["scout"].Effort)
	assert.Equal(t, "economical", effective.Policy.Defaults.Roles["scout"].Capability)
	assert.Equal(t, ".strategist/leveling.yaml", effective.Source)
	assert.Equal(t, 1, effective.DefaultVersion)
	assert.NotEmpty(t, effective.DefaultDigest)
	assert.NotEmpty(t, effective.OverrideDigest)
}

func TestVerifyDefaultAuthorityRejectsStaleManifest(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	require.ErrorContains(t, leveling.VerifyDefaultAuthority(defaults, 1, "stale"), "leveling_policy_stale")
}

func TestSuggestRejectsNonRankedProviderAndUnknownSignal(t *testing.T) {
	policy := defaultPolicy(t)
	policy.Providers["CODEX"] = leveling.Provider{Models: map[string]string{"economical": "x"}, EffortTiers: []string{"medium"}}
	_, err := leveling.Suggest(policy, "CODEX", "scout", leveling.Signals{})
	require.ErrorContains(t, err, "leveling_ranked_provider_ineligible")
	_, err = leveling.Suggest(defaultPolicy(t), "CODEX", "scout", leveling.Signals{Ambiguity: "extreme"})
	require.ErrorContains(t, err, "leveling_signal_unknown")
}

func TestParseRejectsEmptyModelMapping(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	broken := bytes.Replace(defaults, []byte("general: codex-general"), []byte("general: \"\""), 1)
	_, err = leveling.Parse(broken)
	require.ErrorContains(t, err, "leveling_mapping_invalid")
}

func TestParseRejectsUndeclaredCapabilityMapping(t *testing.T) {
	defaults, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	broken := bytes.Replace(defaults, []byte("general: codex-general"), []byte("unknown: codex-general"), 1)
	_, err = leveling.Parse(broken)
	require.ErrorContains(t, err, "undeclared capability")
}

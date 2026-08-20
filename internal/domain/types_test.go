package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActiveConfig_ValidateNoLegacyFields(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.ActiveConfig{}.ValidateNoLegacyFields())

	err := domain.ActiveConfig{ExecutionMode: "sync"}.ValidateNoLegacyFields()
	require.ErrorContains(t, err, "execution_mode")

	err = domain.ActiveConfig{GitPersistenceMode: "auto"}.ValidateNoLegacyFields()
	require.ErrorContains(t, err, "git_persistence_mode")
}

func TestPersonaConfig_ValidateForRuntime(t *testing.T) {
	t.Parallel()
	valid := domain.PersonaConfig{
		ID:            "ranger",
		ToneDirective: "focused",
		PhaseLabels: domain.PhaseLabels{
			Discovery:  "Discovery",
			Refinement: "Refinement",
			Execution:  "Execution",
		},
		Diagnostics: domain.PersonaDiagnostics{
			PipelineHeader:  "header",
			BootstrapOrigin: "origin",
		},
	}
	require.NoError(t, valid.ValidateForRuntime())

	err := domain.PersonaConfig{}.ValidateForRuntime()
	require.ErrorContains(t, err, "id is required")
	require.ErrorContains(t, err, "tone_directive is required")
	require.ErrorContains(t, err, "phase_labels")
	require.ErrorContains(t, err, "diagnostics.pipeline_header")
	require.ErrorContains(t, err, "diagnostics.bootstrap_origin")
}

func TestRoleConfig_Validate(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.RoleConfig{Role: "sniper", Slot: "execution"}.Validate())

	err := domain.RoleConfig{}.Validate()
	require.ErrorContains(t, err, "role is required")
	require.ErrorContains(t, err, "slot is required")

	err = domain.RoleConfig{Role: "sniper", Slot: "bogus"}.Validate()
	require.ErrorContains(t, err, `slot "bogus" is not one of`)
}

func TestRoleSlotMap_Validate(t *testing.T) {
	t.Parallel()
	full := domain.RoleSlotMap{"discovery": "ranger", "refinement": "archivist", "execution": "sniper"}
	require.NoError(t, full.Validate())

	err := domain.RoleSlotMap{}.Validate()
	require.ErrorContains(t, err, "missing slot: discovery")
	require.ErrorContains(t, err, "missing slot: refinement")
	require.ErrorContains(t, err, "missing slot: execution")
}

func TestActiveConfig_Validate(t *testing.T) {
	t.Parallel()
	valid := domain.ActiveConfig{
		Mode:     "epic",
		BasePath: ".analysis",
		Slots: map[string]string{
			"discovery":  "ranger",
			"refinement": "archivist",
			"execution":  "sniper",
		},
	}
	require.NoError(t, valid.Validate())

	err := domain.ActiveConfig{}.Validate()
	require.ErrorContains(t, err, "mode is required")
	require.ErrorContains(t, err, "base_path is required")
	require.ErrorContains(t, err, "slots must have at least one entry")

	err = domain.ActiveConfig{
		Mode:     "epic",
		BasePath: ".analysis",
		Slots:    map[string]string{"discovery": "ranger"},
	}.Validate()
	require.ErrorContains(t, err, "missing slot: refinement")
	require.ErrorContains(t, err, "missing slot: execution")

	err = domain.ActiveConfig{
		Mode:     "epic",
		BasePath: ".analysis",
		Slots: map[string]string{
			"discovery": "ranger", "refinement": "archivist", "execution": "sniper",
		},
		ProviderResolutionPolicy: "bogus",
	}.Validate()
	require.ErrorContains(t, err, `provider_resolution_policy "bogus" is not one of block, ask, native`)
}

func TestResolutionPolicy_Validate(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.ResolutionPolicy("").Validate())
	require.NoError(t, domain.ResolutionPolicyBlock.Validate())
	require.NoError(t, domain.ResolutionPolicyAsk.Validate())
	require.NoError(t, domain.ResolutionPolicyNative.Validate())

	err := domain.ResolutionPolicy("silent").Validate()
	require.ErrorContains(t, err, `provider_resolution_policy "silent" is not one of block, ask, native`)
}

func TestResolutionPolicy_EffectivePolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, domain.ResolutionPolicyAsk, domain.ResolutionPolicy("").EffectivePolicy())
	assert.Equal(t, domain.DefaultResolutionPolicy, domain.ResolutionPolicy("").EffectivePolicy())
	assert.Equal(t, domain.ResolutionPolicyBlock, domain.ResolutionPolicyBlock.EffectivePolicy())
	assert.Equal(t, domain.ResolutionPolicyNative, domain.ResolutionPolicyNative.EffectivePolicy())
}

func TestDecideFallbackOutcome(t *testing.T) {
	t.Parallel()

	// No fallback available: policy is irrelevant, always FallbackOutcomeUnavailable.
	for _, policy := range []domain.ResolutionPolicy{"", domain.ResolutionPolicyBlock, domain.ResolutionPolicyAsk, domain.ResolutionPolicyNative} {
		assert.Equal(t, domain.FallbackOutcomeUnavailable, domain.DecideFallbackOutcome(policy, false),
			"policy=%q fallbackAvailable=false", policy)
	}

	// Fallback available: outcome follows the effective policy's decision table.
	cases := []struct {
		policy domain.ResolutionPolicy
		want   domain.FallbackOutcome
	}{
		{domain.ResolutionPolicyBlock, domain.FallbackOutcomeBlocked},
		{domain.ResolutionPolicyAsk, domain.FallbackOutcomeAskRequired},
		{domain.ResolutionPolicyNative, domain.FallbackOutcomeAutoNative},
		{"", domain.FallbackOutcomeAskRequired}, // empty policy resolves to the "ask" default
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, domain.DecideFallbackOutcome(tc.policy, true), "policy=%q", tc.policy)
	}
}

func TestDecideFallbackOutcome_UnrecognizedPolicyDefaultsToAskRequired(t *testing.T) {
	t.Parallel()
	// An unrecognized policy value (should already be rejected by Validate() at
	// config load time, but this function must still fail toward the more
	// conservative, confirmation-gated outcome rather than silently auto-falling-back).
	assert.Equal(t, domain.FallbackOutcomeAskRequired, domain.DecideFallbackOutcome(domain.ResolutionPolicy("bogus"), true))
}

func TestDecideSlotFallbackOutcome_DiscoveryAlwaysExemptFromPolicy(t *testing.T) {
	t.Parallel()
	for _, policy := range []domain.ResolutionPolicy{domain.ResolutionPolicyBlock, domain.ResolutionPolicyAsk, domain.ResolutionPolicyNative} {
		assert.Equal(t, domain.FallbackOutcomeAlwaysNative, domain.DecideSlotFallbackOutcome("discovery", policy, true),
			"discovery must ignore policy even when a fallback is available, policy=%q", policy)
		assert.Equal(t, domain.FallbackOutcomeAlwaysNative, domain.DecideSlotFallbackOutcome("discovery", policy, false),
			"discovery must ignore policy even when no fallback is available, policy=%q", policy)
	}
}

func TestDecideSlotFallbackOutcome_OtherSlotsDeferToDecideFallbackOutcome(t *testing.T) {
	t.Parallel()
	for _, slot := range []string{"refinement", "execution"} {
		assert.Equal(t, domain.FallbackOutcomeAutoNative, domain.DecideSlotFallbackOutcome(slot, domain.ResolutionPolicyNative, true), "slot=%s", slot)
		assert.Equal(t, domain.FallbackOutcomeBlocked, domain.DecideSlotFallbackOutcome(slot, domain.ResolutionPolicyBlock, true), "slot=%s", slot)
		assert.Equal(t, domain.FallbackOutcomeUnavailable, domain.DecideSlotFallbackOutcome(slot, domain.ResolutionPolicyNative, false), "slot=%s", slot)
	}
}

func TestPersonaConfig_Validate(t *testing.T) {
	t.Parallel()
	require.NoError(t, domain.PersonaConfig{ID: "ranger", ToneDirective: "focused"}.Validate())

	err := domain.PersonaConfig{}.Validate()
	require.ErrorContains(t, err, "id is required")
	require.ErrorContains(t, err, "tone_directive is required")
}

func TestDojoCheckResult_PassedAndFailCount(t *testing.T) {
	t.Parallel()
	allPassed := domain.DojoCheckResult{Items: []domain.DojoCheckItem{{Passed: true}, {Passed: true}}}
	assert.True(t, allPassed.Passed())
	assert.Equal(t, 0, allPassed.FailCount())

	mixed := domain.DojoCheckResult{Items: []domain.DojoCheckItem{{Passed: true}, {Passed: false}, {Passed: false}}}
	assert.False(t, mixed.Passed())
	assert.Equal(t, 2, mixed.FailCount())
}

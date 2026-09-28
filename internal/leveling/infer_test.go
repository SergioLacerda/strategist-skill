package leveling_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderForModel(t *testing.T) {
	policy := defaultPolicy(t)
	assert.Equal(t, "CLAUDE", policy.ProviderForModel("claude-opus-5"), "vendor prefix")
	assert.Equal(t, "CLAUDE", policy.ProviderForModel(" Claude-Sonnet-5 "), "case and space insensitive")
	assert.Empty(t, policy.ProviderForModel("gemini-2"), "an unknown vendor never borrows another provider")
	assert.Empty(t, policy.ProviderForModel(""))
	assert.Equal(t, leveling.ProviderMatchInferred, policy.MatchProvider("claude-opus-5").Match)
	assert.Equal(t, leveling.ProviderMatchExact, policy.MatchProvider("claude-general").Match)
	assert.Equal(t, leveling.ProviderMatchUnknown, policy.MatchProvider("gemini-2").Match)
}

// Prefix inference remains observable but advisory; it cannot silently supply
// an authority-bearing policy effort.
func TestResolveLevelInferredKeepsPrefixMatchAdvisory(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelInferred(loader.load, "archivist", leveling.Signals{}, leveling.Host{Model: "claude-opus-5"})
	require.NoError(t, err)
	assert.Empty(t, level.Provider)
	assert.Empty(t, level.Effort, "prefix inference must not provide policy effort")
	assert.Equal(t, string(leveling.ProviderMatchInferred), level.ProviderMatch)
	assert.Equal(t, leveling.SourceHost, level.ModelSource)
	assert.Equal(t, 1, loader.calls)
}

func TestResolveLevelInferredNeverLoadsThePolicyWithoutNeed(t *testing.T) {
	complete := &countingLoader{policy: defaultPolicy(t)}
	_, err := leveling.ResolveLevelInferred(complete.load, "ranger", leveling.Signals{}, leveling.Host{Model: "claude-opus-5", Effort: "high"})
	require.NoError(t, err)
	assert.Zero(t, complete.calls, "a complete host report never reads the policy")

	noModel := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelInferred(noModel.load, "ranger", leveling.Signals{}, leveling.Host{})
	require.NoError(t, err)
	assert.True(t, level.Unknown())
	assert.Zero(t, noModel.calls, "nothing to infer without a host model")

	unknownVendor := &countingLoader{policy: defaultPolicy(t)}
	level, err = leveling.ResolveLevelInferred(unknownVendor.load, "ranger", leveling.Signals{}, leveling.Host{Model: "gemini-2"})
	require.NoError(t, err)
	assert.Empty(t, level.Effort, "no ranked provider matches: host values only")
	assert.Equal(t, "Gemini-2", level.Model)
}

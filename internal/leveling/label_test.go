package leveling_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayNameUsesConfiguredNameThenTitleCase(t *testing.T) {
	policy := defaultPolicy(t)
	provider := policy.Providers["CLAUDE"]
	provider.Display = map[string]string{"claude-reasoning": "Opus"}
	policy.Providers["CLAUDE"] = provider

	assert.Equal(t, "Opus", policy.DisplayName("claude", "claude-reasoning"))
	assert.Equal(t, "General", policy.DisplayName("CLAUDE", "claude-general"))
	assert.Equal(t, "Sonnet", policy.DisplayName("CLAUDE", "sonnet"))
	assert.Empty(t, policy.DisplayName("CLAUDE", ""))
}

func TestMergeKeepsDefaultDisplayAndOverlaysCustomer(t *testing.T) {
	defaults := []byte(`version: 1
defaults:
  effort_tiers: [low, medium, high]
  fallback: {capability: general, effort: medium, reason: generic}
  roles:
    ranger: {capability: reasoning, effort: high, criteria: {ambiguity: low, risk: low, scope: bounded, evidence: sufficient}, escalation: {capability: reasoning, effort: high}}
providers:
  CLAUDE:
    ranked: true
    models: {general: claude-general, economical: claude-economical, reasoning: claude-reasoning}
    display: {claude-general: Sonnet}
    effort_tiers: [low, medium, high]
`)
	override := []byte("providers:\n  CLAUDE:\n    display: {claude-reasoning: Opus}\n")
	policy, err := leveling.Merge(defaults, override)
	require.NoError(t, err)
	assert.Equal(t, "Sonnet", policy.DisplayName("CLAUDE", "claude-general"))
	assert.Equal(t, "Opus", policy.DisplayName("CLAUDE", "claude-reasoning"))
}

func TestValidateRejectsBlankDisplayName(t *testing.T) {
	policy := defaultPolicy(t)
	provider := policy.Providers["CLAUDE"]
	provider.Display = map[string]string{"claude-reasoning": " "}
	policy.Providers["CLAUDE"] = provider
	err := policy.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leveling_mapping_invalid")
}

func TestResolveLevelHostWinsOverPolicy(t *testing.T) {
	policy := defaultPolicy(t)
	level, err := leveling.ResolveLevel(policy, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{Model: "sonnet", Effort: "high"})
	require.NoError(t, err)
	assert.Equal(t, leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost, ModelSource: leveling.SourceHost, EffortSource: leveling.SourceHost}, level)
	assert.Equal(t, "Sonnet-High", level.Label())
}

func TestResolveLevelFallsBackToPolicy(t *testing.T) {
	policy := defaultPolicy(t)
	level, err := leveling.ResolveLevel(policy, "CLAUDE", "archivist", leveling.Signals{}, leveling.Host{})
	require.NoError(t, err)
	assert.Equal(t, leveling.SourcePolicy, level.Source)
	assert.Equal(t, "archivist", level.Role)
	assert.Equal(t, "high", level.Effort)
	assert.NotEmpty(t, level.Model)
}

func TestResolveLevelHostPartialCompletesFromPolicy(t *testing.T) {
	policy := defaultPolicy(t)
	level, err := leveling.ResolveLevel(policy, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{Model: "Opus"})
	require.NoError(t, err)
	assert.Equal(t, "Opus", level.Model)
	assert.Equal(t, "high", level.Effort)
	assert.Equal(t, leveling.SourceHost, level.Source)
}

func TestResolveLevelUnknownWithoutProviderOrHost(t *testing.T) {
	level, err := leveling.ResolveLevel(defaultPolicy(t), "", "ranger", leveling.Signals{}, leveling.Host{})
	require.NoError(t, err)
	assert.True(t, level.Unknown())
	assert.Empty(t, level.Label())
}

func TestResolveLevelHostOnlyNeedsNoProvider(t *testing.T) {
	level, err := leveling.ResolveLevel(defaultPolicy(t), "", "sniper", leveling.Signals{}, leveling.Host{Model: "haiku", Effort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Haiku-Low", level.Label())
}

func TestResolveLevelSurfacesPolicyErrors(t *testing.T) {
	_, err := leveling.ResolveLevel(defaultPolicy(t), "CLAUDE", "ranger", leveling.Signals{Risk: "bogus"}, leveling.Host{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leveling_signal_unknown")
}

func TestRenderLayouts(t *testing.T) {
	level := leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost}
	stacked := "Fase: 01/04\nRanger\nSonnet-High\nmessage A"
	inline := "Ranger(Sonnet-High) - message A"

	assert.Equal(t, stacked, leveling.Render(level, "message A", 0), "unmeasurable width stacks")
	assert.Equal(t, inline, leveling.Render(level, "message A", 80), "fits inline")
	assert.Equal(t, stacked, leveling.Render(level, "message A", 20), "too narrow stacks")
}

func TestRenderPhaseCounterPerRole(t *testing.T) {
	cases := map[string]string{
		"scout": "Fase: 00/04", "ranger": "Fase: 01/04", "archivist": "Fase: 02/04",
		"gate": "Fase: 03/04", "sniper": "Fase: 04/04",
	}
	for role, want := range cases {
		out := leveling.Render(leveling.Level{Role: role, Model: "M", Effort: "low"}, "x", 0)
		assert.Contains(t, out, want, role)
	}
	transport := leveling.Render(leveling.Level{Role: "transport", Model: "M", Effort: "low"}, "x", 0)
	assert.NotContains(t, transport, "Fase:")
	assert.Equal(t, "Transport\nM-Low\nx", transport)
}

func TestRenderUnknownLevelKeepsUnlabelledFormat(t *testing.T) {
	out := leveling.Render(leveling.Level{Role: "ranger"}, "message", 80)
	assert.Equal(t, "Ranger - message", out)
}

func TestLedgerRecordsAndReusesLatestPerMissionRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	_, ok, err := leveling.LatestRecord(path, "m1", "ranger")
	require.NoError(t, err)
	assert.False(t, ok, "missing ledger is not an error")

	first := leveling.Record{MissionID: "m1", Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost}}
	require.NoError(t, leveling.AppendRecord(path, first))
	escalated := leveling.Record{MissionID: "m1", Reason: "escalated", Level: leveling.Level{Role: "ranger", Model: "Opus", Effort: "xhigh", Source: leveling.SourceHost}}
	require.NoError(t, leveling.AppendRecord(path, escalated))
	other := leveling.Record{MissionID: "m2", Level: leveling.Level{Role: "ranger", Model: "Haiku", Effort: "low", Source: leveling.SourcePolicy}}
	require.NoError(t, leveling.AppendRecord(path, other))

	got, ok, err := leveling.LatestRecord(path, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Opus", got.Model)
	assert.Equal(t, "escalated", got.Reason)
	assert.NotEmpty(t, got.Timestamp)
}

func TestLedgerIgnoresMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	require.NoError(t, leveling.AppendRecord(path, leveling.Record{MissionID: "m1", Level: leveling.Level{Role: "sniper", Model: "Haiku", Effort: "low", Source: leveling.SourcePolicy}}))
	appendRaw(t, path, "{not json\n")
	got, ok, err := leveling.LatestRecord(path, "m1", "sniper")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Haiku", got.Model)
}

func appendRaw(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // test temp file
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestDisplayNamesAreNotPartOfThePolicyIdentity(t *testing.T) {
	policy := defaultPolicy(t)
	before := policy.Digest()

	// Presentation must never invalidate an installed workspace: changing or
	// removing display names leaves the digest untouched.
	provider := policy.Providers["CLAUDE"]
	provider.Display = map[string]string{"claude-reasoning": "Something else"}
	policy.Providers["CLAUDE"] = provider
	assert.Equal(t, before, policy.Digest())
	provider.Display = nil
	policy.Providers["CLAUDE"] = provider
	assert.Equal(t, before, policy.Digest())

	encoded, err := json.Marshal(policy)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "Display")
}

func TestEmbeddedDefaultsShipRealClaudeDisplayNames(t *testing.T) {
	policy := defaultPolicy(t)
	assert.Equal(t, "Sonnet", policy.DisplayName("CLAUDE", "claude-general"))
	assert.Equal(t, "Haiku", policy.DisplayName("CLAUDE", "claude-economical"))
	assert.Equal(t, "Opus", policy.DisplayName("CLAUDE", "claude-reasoning"))
	assert.Equal(t, "Reasoning", policy.DisplayName("CODEX", "codex-reasoning"), "providers without display keep the derived name")
}

func TestPolicyLabelUsesShippedDisplayNames(t *testing.T) {
	level, err := leveling.ResolveLevel(defaultPolicy(t), "CLAUDE", "archivist", leveling.Signals{}, leveling.Host{})
	require.NoError(t, err)
	assert.Equal(t, "Opus-High", level.Label())
}

type countingLoader struct {
	calls  int
	policy leveling.Policy
	err    error
}

func (c *countingLoader) load() (leveling.Policy, error) {
	c.calls++
	return c.policy, c.err
}

func TestResolveLevelLazyHostCompleteNeverLoadsPolicy(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelLazy(loader.load, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{Model: "Opus", Effort: "low"})
	require.NoError(t, err)
	assert.Equal(t, leveling.SourceHost, level.Source)
	assert.Zero(t, loader.calls)
}

func TestResolveLevelLazyLoadsPolicyOnlyForMissingValues(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelLazy(loader.load, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{Model: "Opus"})
	require.NoError(t, err)
	assert.Equal(t, "Opus", level.Model, "host model wins")
	assert.Equal(t, "high", level.Effort, "effort completed from the policy")
	assert.Equal(t, leveling.SourceHost, level.Source, "source names the highest-precedence contributor")
	assert.Equal(t, 1, loader.calls)

	auto := &countingLoader{policy: defaultPolicy(t)}
	level, err = leveling.ResolveLevelLazy(auto.load, "CLAUDE", "archivist", leveling.Signals{}, leveling.Host{})
	require.NoError(t, err)
	assert.Equal(t, leveling.SourcePolicy, level.Source)
	assert.Equal(t, 1, auto.calls)
}

func TestResolveLevelLazyWithoutProviderNeverLoadsPolicy(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelLazy(loader.load, "", "ranger", leveling.Signals{}, leveling.Host{Effort: "high"})
	require.NoError(t, err)
	assert.Equal(t, "High", level.Label())
	assert.Zero(t, loader.calls)
}

func TestResolveLevelLazySurfacesLoaderError(t *testing.T) {
	loader := &countingLoader{err: assert.AnError}
	_, err := leveling.ResolveLevelLazy(loader.load, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{})
	require.ErrorIs(t, err, assert.AnError)
}

func TestRenderWithRegistryDerivesTotalAndPhase(t *testing.T) {
	roles := domain.DefaultRoleRegistry().Roles()
	roles = append(roles, domain.Role{ID: "auditor", Phase: 5})
	reg, err := domain.NewRoleRegistry(roles)
	require.NoError(t, err)

	out := leveling.RenderWith(reg, leveling.Level{Role: "auditor", Model: "M", Effort: "low"}, "x", 0)
	assert.Equal(t, "Fase: 05/05\nAuditor\nM-Low\nx", out)
	gate := leveling.RenderWith(reg, leveling.Level{Role: "gate", Model: "M", Effort: "low"}, "x", 0)
	assert.Contains(t, gate, "Fase: 03/05", "the gate still sits before the execution role")
}

func TestPolicyRoleUsesTheRoleLevelingKey(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{{ID: "ranger", Slot: "discovery", Phase: 1, Pluggable: true, Leveling: "archivist"}})
	require.NoError(t, err)
	policy := defaultPolicy(t)
	got, err := leveling.Suggest(policy, "CLAUDE", reg.PolicyRole("ranger"), leveling.Signals{})
	require.NoError(t, err)
	want, err := leveling.Suggest(policy, "CLAUDE", "archivist", leveling.Signals{})
	require.NoError(t, err)
	assert.Equal(t, want.Effort, got.Effort)
	assert.Equal(t, want.Capability, got.Capability)
}

func TestLedgerKeysReuseByPhaseRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	first := leveling.Record{MissionID: "m1", Level: leveling.Level{Role: "archivist", Model: "Sonnet", Effort: "medium", Source: leveling.SourceHost}}
	require.NoError(t, leveling.AppendRecord(path, first))
	revision := leveling.Record{MissionID: "m1", Run: "2", Level: leveling.Level{Role: "archivist", Model: "Opus", Effort: "high", Source: leveling.SourceHost}}
	require.NoError(t, leveling.AppendRecord(path, revision))

	got, ok, err := leveling.LatestRunRecord(path, "m1", "archivist", "")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Sonnet", got.Model, "the default run keeps its own tuple")

	got, ok, err = leveling.LatestRunRecord(path, "m1", "archivist", "2")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Opus", got.Model)
	assert.Equal(t, "2", got.Run)

	_, ok, err = leveling.LatestRunRecord(path, "m1", "archivist", "3")
	require.NoError(t, err)
	assert.False(t, ok, "an unseen run has no recorded level yet")

	legacy, ok, err := leveling.LatestRecord(path, "m1", "archivist")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Sonnet", legacy.Model, "LatestRecord is the default run")
}

func TestLevelTagIsTheInlineForm(t *testing.T) {
	assert.Equal(t, "(Sonnet-High)", leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high"}.Tag())
	assert.Equal(t, "(Opus)", leveling.Level{Role: "ranger", Model: "Opus"}.Tag())
	assert.Empty(t, leveling.Level{Role: "ranger"}.Tag(), "an unknown level leaves the role name unlabelled")
}

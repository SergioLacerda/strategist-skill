package leveling

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostSanitizationAndHelpers(t *testing.T) {
	assert.True(t, ValidEffortTier("high"))
	assert.True(t, ValidEffortTier("  HIGH  "))
	assert.False(t, ValidEffortTier("ultra"))

	names := EffortTierNames()
	assert.NotEmpty(t, names)

	// Clean host inputs
	h, rejected := SanitizeHost(Host{Model: "gpt-4", Effort: "high"})
	assert.Empty(t, rejected)
	assert.Equal(t, "gpt-4", h.Model)
	assert.Equal(t, "high", h.Effort)

	// Placeholders and multi-line
	hPlaceholders, rejectedP := SanitizeHost(Host{Model: "<your-model>", Effort: "<your-effort>"})
	assert.Len(t, rejectedP, 2)
	assert.Empty(t, hPlaceholders.Model)
	assert.Empty(t, hPlaceholders.Effort)

	hMulti, rejectedM := SanitizeHost(Host{Model: "gpt-4\nsecond-line", Effort: "invalid-tier"})
	assert.Len(t, rejectedM, 2)
	assert.Empty(t, hMulti.Model)
	assert.Empty(t, hMulti.Effort)
}

func TestLedgerOperationsAndCorruptedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "role-levels.jsonl")

	// Append first record
	rec1 := Record{MissionID: "m1", Level: Level{Role: "Ranger", Model: "codex-1", Effort: "high"}}
	require.NoError(t, AppendRecord(path, rec1))

	// Append corrupt line
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("corrupted json line\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Append second record
	rec2 := Record{MissionID: "m1", Level: Level{Role: "ranger", Model: "codex-2", Effort: "high"}}
	require.NoError(t, AppendRecord(path, rec2))

	latest, found, err := LatestRecord(path, "m1", "RANGER")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "codex-2", latest.Model)

	// Nonexistent mission
	_, foundNone, err := LatestRecord(path, "missing-mission", "ranger")
	require.NoError(t, err)
	assert.False(t, foundNone)

	// Nonexistent file
	_, foundNoFile, err := LatestRecord(filepath.Join(dir, "no-file"), "m1", "ranger")
	require.NoError(t, err)
	assert.False(t, foundNoFile)
}

func TestLoadEffectiveAndDefaultAuthority(t *testing.T) {
	defaults := []byte(`
version: 1
defaults:
  effort_tiers: ["low", "medium", "high"]
  fallback:
    capability: reasoning
    effort: medium
    reason: fallback
  roles:
    ranger:
      capability: reasoning
      effort: medium
providers:
  CODEX:
    ranked: true
    models:
      reasoning: gpt-4
    effort_tiers: ["low", "medium", "high"]
`)
	eff, err := LoadEffective(defaults, nil, "test")
	require.NoError(t, err)
	assert.Equal(t, 1, eff.DefaultVersion)
	assert.Empty(t, eff.OverrideDigest)

	// VerifyDefaultAuthority mismatch
	errStale := VerifyDefaultAuthority(defaults, 99, "wrong-digest")
	require.ErrorContains(t, errStale, "leveling_policy_stale")

	// VerifyDefaultAuthority match
	require.NoError(t, VerifyDefaultAuthority(defaults, 1, eff.DefaultDigest))

	// CheckDefaultParity mismatch
	diffDefaults := []byte(`
version: 2
defaults:
  effort_tiers: ["low", "medium", "high"]
  fallback:
    capability: reasoning
    effort: medium
    reason: fallback
  roles:
    ranger:
      capability: reasoning
      effort: medium
providers:
  CODEX:
    ranked: true
    models:
      reasoning: gpt-4
    effort_tiers: ["low", "medium", "high"]
`)
	errParity := CheckDefaultParity(defaults, diffDefaults)
	require.ErrorContains(t, errParity, "digest mismatch")
}

func TestReadOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leveling.yaml")

	// Missing file
	data, exists, err := ReadOverride(path)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, data)

	// Existing file
	require.NoError(t, os.WriteFile(path, []byte("version: 1\n"), 0o600))
	data, exists, err = ReadOverride(path)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NotEmpty(t, data)

	// Read error (directory)
	dirPath := filepath.Join(dir, "dir_override")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	_, _, errDir := ReadOverride(dirPath)
	require.Error(t, errDir)
	assert.Contains(t, errDir.Error(), ReasonLevelingPolicyUnreadable)
}

func TestPolicyValidationEdgeCases(t *testing.T) {
	p := Policy{
		Version: 0,
	}
	require.ErrorContains(t, p.Validate(), "version must be positive")

	baseYAML := []byte(`
version: 1
defaults:
  effort_tiers: ["low", "medium", "high"]
  fallback:
    capability: reasoning
    effort: medium
    reason: fallback
  roles:
    ranger:
      capability: reasoning
      effort: medium
      criteria:
        ambiguity: invalid_ambiguity
providers:
  CODEX:
    ranked: true
    models:
      reasoning: gpt-4
    effort_tiers: ["low", "medium", "high"]
`)
	_, err := Parse(baseYAML)
	require.ErrorContains(t, err, "leveling_signal_unknown")
}

func TestRotateLedgerOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "role-levels.jsonl")

	// Nonexistent ledger rotation
	dropped, err := RotateLedger(path, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, dropped)

	droppedIfLarge, err := RotateLedgerIfLarge(path, 10, 100)
	require.NoError(t, err)
	assert.Equal(t, 0, droppedIfLarge)

	// Append multiple records for same mission and role
	for i := 1; i <= 5; i++ {
		require.NoError(t, AppendRecord(path, Record{MissionID: "m1", Level: Level{Role: "ranger", Model: "m", Effort: "high"}}))
	}

	// Rotate with maxRecords = 2 (should keep latest record + 1 older = 2 records, dropping 3)
	droppedRot, err := RotateLedger(path, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, droppedRot)

	// RotateIfLarge when small
	droppedSmall, err := RotateLedgerIfLarge(path, 100000, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, droppedSmall)
}

func TestSuggestEscalationsAndErrors(t *testing.T) {
	policyRaw := []byte(`
version: 1
defaults:
  effort_tiers: ["low", "medium", "high"]
  fallback:
    capability: reasoning
    effort: medium
    reason: fallback
  roles:
    ranger:
      capability: reasoning
      effort: medium
      criteria:
        ambiguity: low
        risk: low
        scope: bounded
        evidence: sufficient
      escalation:
        capability: deep_reasoning
        effort: high
providers:
  CODEX:
    ranked: true
    models:
      reasoning: gpt-4
      deep_reasoning: o3-mini
    effort_tiers: ["low", "medium", "high"]
  UNRANKED:
    ranked: false
    models:
      reasoning: custom
    effort_tiers: ["low"]
`)
	policy, err := Parse(policyRaw)
	require.NoError(t, err)

	// Invalid signals
	_, errSig := Suggest(policy, "CODEX", "ranger", Signals{Ambiguity: "invalid"})
	require.ErrorContains(t, errSig, "leveling_signal_unknown")

	// Empty provider
	_, errEmp := Suggest(policy, "", "ranger", Signals{})
	require.ErrorContains(t, errEmp, "provider identifier must not be empty")

	// Provider unranked
	_, errUnranked := Suggest(policy, "UNRANKED", "ranger", Signals{})
	require.ErrorContains(t, errUnranked, "is not ranked")

	// Generic provider fallback (unknown provider)
	sugFall, err := Suggest(policy, "UNKNOWN_PROVIDER", "ranger", Signals{})
	require.NoError(t, err)
	assert.True(t, sugFall.FallbackUsed)
	assert.Contains(t, sugFall.Rationale, "generic_provider_fallback")

	// Normal suggestion
	sugNorm, err := Suggest(policy, "CODEX", "ranger", Signals{})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", sugNorm.Model)
	assert.Equal(t, "reasoning", sugNorm.Capability)

	// Escalations: SecuritySensitive, ArchitecturalChange, RepeatedFailures, High Ambiguity, High Risk, Cross Module, Conflicting Evidence
	for _, sig := range []Signals{
		{SecuritySensitive: true},
		{ArchitecturalChange: true},
		{RepeatedFailures: 2},
		{Ambiguity: "high"},
		{Risk: "high"},
		{Scope: "cross_module"},
		{Evidence: "conflicting"},
	} {
		sugEsc, err := Suggest(policy, "CODEX", "ranger", sig)
		require.NoError(t, err)
		assert.Equal(t, "o3-mini", sugEsc.Model)
		assert.Equal(t, "deep_reasoning", sugEsc.Capability)
	}
}

package leveling_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B2: the role id is normalized at the ledger boundary, so every spelling is
// one role for reuse and reporting.
func TestLedgerNormalizesRoleOnWriteAndLookup(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	require.NoError(t, leveling.AppendRecord(ledger, leveling.Record{MissionID: "m1", Level: leveling.Level{Role: " Ranger ", Model: "Opus", Effort: "high"}}))

	record, ok, err := leveling.LatestRecord(ledger, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok, "a mixed-case write must be found by the canonical id")
	assert.Equal(t, "ranger", record.Role)

	_, ok, err = leveling.LatestRecord(ledger, "m1", "RANGER")
	require.NoError(t, err)
	assert.True(t, ok, "lookup normalizes the requested role too")
}

func TestLegacyMixedCaseLedgerLinesSummarizeAsOneRole(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	legacy := `{"mission_id":"m1","role":"Ranger","model":"Opus","effort":"high","level_source":"host","timestamp":"2026-09-21T00:00:00Z"}
{"mission_id":"m1","role":"ranger","model":"Opus","effort":"high","level_source":"host","timestamp":"2026-09-21T00:00:01Z"}
`
	require.NoError(t, os.WriteFile(ledger, []byte(legacy), 0o600))

	records, err := leveling.ReadRecords(ledger)
	require.NoError(t, err)
	report := leveling.Summarize(records)
	require.Len(t, report.Roles, 1, "legacy `Ranger` and `ranger` must not be two roles")
	assert.Equal(t, "ranger", report.Roles[0].Role)
	assert.Equal(t, 2, report.Roles[0].Records)
}

// H1 + H2: untrusted host input is dropped with an explanation.
func TestSanitizeHost(t *testing.T) {
	cases := []struct {
		name         string
		in, want     leveling.Host
		rejectedHint []string
	}{
		{name: "valid values pass and effort is lower-cased", in: leveling.Host{Model: " claude-sonnet-5 ", Effort: "High"}, want: leveling.Host{Model: "claude-sonnet-5", Effort: "high"}},
		{name: "empty stays empty", in: leveling.Host{}, want: leveling.Host{}},
		{name: "unreplaced on_start placeholders", in: leveling.Host{Model: "<your-model>", Effort: "<your-effort>"}, want: leveling.Host{}, rejectedHint: []string{"placeholder", "placeholder"}},
		{name: "effort outside the tier catalog", in: leveling.Host{Model: "Opus", Effort: "banana"}, want: leveling.Host{Model: "Opus"}, rejectedHint: []string{"not one of"}},
		{name: "multi-line model", in: leveling.Host{Model: "Opus\nx", Effort: "low"}, want: leveling.Host{Effort: "low"}, rejectedHint: []string{"single line"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, rejected := leveling.SanitizeHost(tc.in)
			assert.Equal(t, tc.want, got)
			require.Len(t, rejected, len(tc.rejectedHint))
			for i, hint := range tc.rejectedHint {
				assert.Contains(t, rejected[i], hint)
			}
		})
	}
}

func TestEffortTierNamesMatchValidation(t *testing.T) {
	for _, tier := range leveling.EffortTierNames() {
		assert.True(t, leveling.ValidEffortTier(tier), tier)
	}
	assert.False(t, leveling.ValidEffortTier("banana"))
}

// H4: a host model id is shortened without loading the policy when the host
// answered completely, and through the configured display table otherwise.
func TestHostModelUsesDisplayFallbackWithoutLoadingPolicy(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelLazy(loader.load, "CLAUDE", "ranger", leveling.Signals{}, leveling.Host{Model: "claude-sonnet-5", Effort: "medium"})
	require.NoError(t, err)
	assert.Equal(t, "Sonnet-5-Medium", level.Label(), "the provider prefix is dropped without reading the policy")
	assert.Zero(t, loader.calls, "a complete host report must still never read the policy")
}

func TestPartialHostModelUsesConfiguredDisplayName(t *testing.T) {
	loader := &countingLoader{policy: defaultPolicy(t)}
	level, err := leveling.ResolveLevelLazy(loader.load, "CLAUDE", "archivist", leveling.Signals{}, leveling.Host{Model: "claude-general"})
	require.NoError(t, err)
	assert.Equal(t, "Sonnet", level.Model, "a host model is mapped through the policy display table once the policy is loaded")
	assert.Equal(t, leveling.SourceHost, level.ModelSource)
	assert.Equal(t, 1, loader.calls)
}

func TestModelWithoutProviderIsOnlyCapitalized(t *testing.T) {
	level, err := leveling.ResolveLevelLazy((&countingLoader{}).load, "", "ranger", leveling.Signals{}, leveling.Host{Model: "claude-sonnet-5", Effort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Claude-sonnet-5", level.Model, "with no provider there is no prefix to strip")
}

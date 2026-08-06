package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFilterRowsByScope_EmptyValueIsNoop(t *testing.T) {
	t.Parallel()
	rows := []StatusRow{{ID: "a", Scope: []string{"discovery"}}}
	assert.Equal(t, rows, FilterRowsByScope(rows, ""))
}

func TestFilterRowsByScope_MatchesAllScope(t *testing.T) {
	t.Parallel()
	rows := []StatusRow{{ID: "a", Scope: []string{"all"}}}
	assert.Len(t, FilterRowsByScope(rows, "execution"), 1)
}

func TestFilterRowsByScope_ExcludesUnscopedRows(t *testing.T) {
	t.Parallel()
	rows := []StatusRow{{ID: "a", Scope: nil}}
	assert.Empty(t, FilterRowsByScope(rows, "discovery"))
}

func TestDeriveFreshness_WithLastReviewed(t *testing.T) {
	t.Parallel()
	r := StatusRow{LastReviewed: "2026-06-24"}
	assert.Equal(t, "fresh", DeriveFreshness(r))
}

func TestDeriveFreshness_WithoutLastReviewed(t *testing.T) {
	t.Parallel()
	r := StatusRow{}
	assert.Equal(t, "unknown", DeriveFreshness(r))
}

func TestDeriveDrift_MissingGovernance(t *testing.T) {
	t.Parallel()
	r := StatusRow{Configured: true, Governed: false, Indexed: true}
	drift := DeriveDrift(r)
	assert.Contains(t, drift, "missing_governance")
}

func TestDeriveDrift_MissingIndex(t *testing.T) {
	t.Parallel()
	r := StatusRow{Configured: true, Governed: true, Indexed: false}
	drift := DeriveDrift(r)
	assert.Contains(t, drift, "missing_index")
}

func TestDeriveDrift_Unscoped(t *testing.T) {
	t.Parallel()
	r := StatusRow{Configured: false, Governed: true, Indexed: true}
	drift := DeriveDrift(r)
	assert.Contains(t, drift, "unscoped")
}

func TestDeriveDrift_None(t *testing.T) {
	t.Parallel()
	r := StatusRow{Configured: true, Governed: true, Indexed: true}
	drift := DeriveDrift(r)
	assert.Empty(t, drift)
}

func TestScopeVal_UnmarshalScalar(t *testing.T) {
	t.Parallel()
	input := []byte("scope: all\n")
	var out struct {
		Scope Scope `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"all"}, []string(out.Scope))
}

func TestScopeVal_UnmarshalList(t *testing.T) {
	t.Parallel()
	input := []byte("scope:\n  - discovery\n  - refinement\n")
	var out struct {
		Scope Scope `yaml:"scope"`
	}
	require.NoError(t, yaml.Unmarshal(input, &out))
	assert.Equal(t, []string{"discovery", "refinement"}, []string(out.Scope))
}

func TestMergeChestRows_GovernedNotInActive(t *testing.T) {
	t.Parallel()
	governed := map[string]GovernedChest{
		"gov-only": {ID: "gov-only", Path: "/some/path", Trust: GovernedTrust{Tier: "T1"}},
	}
	rows := MergeChestRows(nil, governed, nil, nil, nil)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "gov-only", r.ID)
	assert.True(t, r.Governed)
	assert.False(t, r.Configured)
	assert.Contains(t, r.Drift, "unscoped")
}

func TestMergeChestRows_IndexedNotDeclared(t *testing.T) {
	t.Parallel()
	indexed := map[string]bool{"idx-only": true}
	rows := MergeChestRows(nil, nil, indexed, nil, nil)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "idx-only", r.ID)
	assert.True(t, r.Indexed)
	assert.False(t, r.Governed)
	assert.False(t, r.Configured)
	assert.Equal(t, "unknown", r.Freshness)
	assert.Contains(t, r.Drift, "unscoped")
}

func TestMergeChestRows_FullMerge(t *testing.T) {
	t.Parallel()
	active := []ActiveChest{
		{ID: "chest-a", Path: "/a", Scope: []string{"discovery"}},
	}
	governed := map[string]GovernedChest{
		"chest-a":  {ID: "chest-a", Trust: GovernedTrust{Tier: "T1", LastReviewed: "2026-01-01"}},
		"gov-only": {ID: "gov-only", Path: "/b", Trust: GovernedTrust{Tier: "T2"}},
	}
	indexed := map[string]bool{"chest-a": true, "idx-only": true}
	compiled := map[string]bool{"chest-a": true}

	rows := MergeChestRows(active, governed, indexed, compiled, nil)
	assert.Len(t, rows, 3)

	byID := make(map[string]StatusRow)
	for _, r := range rows {
		byID[r.ID] = r
	}

	a := byID["chest-a"]
	assert.True(t, a.Configured)
	assert.True(t, a.Governed)
	assert.True(t, a.Indexed)
	assert.True(t, a.Compiled)
	assert.Equal(t, "fresh", a.Freshness)
	assert.Empty(t, a.Drift)

	govOnly := byID["gov-only"]
	assert.False(t, govOnly.Configured)
	assert.True(t, govOnly.Governed)
	assert.Contains(t, govOnly.Drift, "unscoped")

	idxOnly := byID["idx-only"]
	assert.False(t, idxOnly.Configured)
	assert.True(t, idxOnly.Indexed)
}

func TestHistoricalCount(t *testing.T) {
	t.Parallel()
	rows := []StatusRow{
		{ID: "a", TrustTier: "T0"},
		{ID: "b", TrustTier: "T1"},
		{ID: "c", TrustTier: "T2"},
		{ID: "d", TrustTier: "T3"},
	}
	assert.Equal(t, 2, HistoricalCount(rows))
	assert.Equal(t, 0, HistoricalCount(nil))
}

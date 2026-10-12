package calibration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEverySyntheticPairIsValidAndKeepsItsLabelOutOfTheState(t *testing.T) {
	pairs := SyntheticPairs(12)
	require.Len(t, pairs, 72)
	ids := map[string]bool{}
	for _, p := range pairs {
		require.NoError(t, p.Validate(), p.ID)
		assert.False(t, ids[p.ID], "duplicate id %s", p.ID)
		ids[p.ID] = true
		state := p.State()
		assert.NotContains(t, state, p.Family)
		assert.NotContains(t, state, p.ID)
		assert.NotContains(t, state, p.Parent)
		assert.NotContains(t, state, "oracle")
	}
}

func brokenDimensions(p Pair) []string {
	falseDims := []string{}
	for dim, holds := range p.Oracle {
		if !holds {
			falseDims = append(falseDims, dim)
		}
	}
	return falseDims
}

func TestEachMutantBreaksExactlyOneDimension(t *testing.T) {
	broken := map[string]string{FamilyWeakened: DimObligation, FamilyWrongKind: DimKind, FamilyWrongCheck: DimVerification}
	for _, p := range SyntheticPairs(3) {
		want := []string{}
		if dim, ok := broken[p.Family]; ok {
			want = []string{dim}
		}
		assert.Equal(t, want, brokenDimensions(p), p.ID)
	}
}

func TestSyntheticPairsAreDeterministicAndSortedAndCapped(t *testing.T) {
	a, b := SyntheticPairs(4), SyntheticPairs(4)
	assert.Equal(t, a, b)
	assert.Len(t, SyntheticPairs(1000), 72)
	assert.Len(t, SyntheticPairs(0), 6)
	assert.Less(t, a[0].ID, a[1].ID)
}

func TestAPairThatNamesItsLabelIsRefused(t *testing.T) {
	p := SyntheticPairs(1)[0]
	p.Target.Statement += " (" + p.Family + ")"
	require.ErrorContains(t, p.Validate(), "names its own label")

	bad := SyntheticPairs(1)[0]
	bad.Oracle = map[string]bool{"nonsense": true}
	require.Error(t, bad.Validate())
	bad = Pair{}
	require.Error(t, bad.Validate())
}

func TestAMissingVerificationIsShownAsAbsentAndTheSplitFollowsTheParent(t *testing.T) {
	p := SyntheticPairs(1)[0]
	p.Target.Verification = ""
	assert.Contains(t, p.State(), "verification: (absent)")
	first := PairHeldOut("P01", "s", 30)
	assert.Equal(t, first, PairHeldOut("P01", "s", 30))
	assert.False(t, PairHeldOut("P01", "s", 0))
}

package treasure

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- query ---

func TestFilterPotions_DefaultExcludesDeprecated(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {
			{ID: "potion-1", ChestID: "runbooks", Status: domain.PotionStatusDeprecated},
			{ID: "potion-2", ChestID: "runbooks", Status: domain.PotionStatusProposed},
		},
	}

	got := FilterPotions(potions, PotionFilter{})
	require.Len(t, got, 1)
	assert.Equal(t, "potion-2", got[0].ID)
}

func TestFindPotion(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {{ID: "potion-1", ChestID: "runbooks"}},
	}

	got, ok := FindPotion(potions, "potion-1")
	require.True(t, ok)
	assert.Equal(t, "runbooks", got.ChestID)

	_, ok = FindPotion(potions, "missing")
	assert.False(t, ok)
}

func TestFilterPotions_AllIncludesDeprecated(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {
			{ID: "potion-1", ChestID: "runbooks", Status: domain.PotionStatusDeprecated},
		},
	}

	got := FilterPotions(potions, PotionFilter{Status: "all"})
	require.Len(t, got, 1)
	assert.Equal(t, "potion-1", got[0].ID)
}

func TestFilterPotions_ChestAndStatus(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"a": {{ID: "potion-a", ChestID: "a", Status: domain.PotionStatusProposed}},
		"b": {{ID: "potion-b", ChestID: "b", Status: domain.PotionStatusProposed}},
	}

	got := FilterPotions(potions, PotionFilter{
		ChestID: "b",
		Status:  domain.PotionStatusProposed,
	})

	require.Len(t, got, 1)
	assert.Equal(t, "potion-b", got[0].ID)
}

func TestSortPotions(t *testing.T) {
	t.Parallel()
	potions := []Potion{
		{ID: "potion-2", ChestID: "b"},
		{ID: "potion-1", ChestID: "a"},
		{ID: "potion-3", ChestID: "a"},
	}

	SortPotions(potions)

	assert.Equal(t, []string{"potion-1", "potion-3", "potion-2"}, []string{potions[0].ID, potions[1].ID, potions[2].ID})
}

func TestProposedPotions(t *testing.T) {
	t.Parallel()
	potions := map[string][]Potion{
		"runbooks": {
			{ID: "potion-1", ChestID: "runbooks", Status: domain.PotionStatusProposed},
			{ID: "potion-2", ChestID: "runbooks", Status: domain.PotionStatusAccepted},
		},
	}

	got := ProposedPotions(potions)
	require.Len(t, got, 1)
	assert.Equal(t, "potion-1", got[0].ID)
}

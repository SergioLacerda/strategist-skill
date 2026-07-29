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

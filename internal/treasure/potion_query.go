package treasure

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// --- query ---

// PotionFilter selects potions for read-only listing.
type PotionFilter struct {
	ChestID string
	Status  string
}

// FilterPotions flattens grouped potions, applies the requested filter, and returns a
// deterministic chest/id ordering for CLI and JSON output. Mirrors FilterJewels.
func FilterPotions(potionsByChest map[string][]Potion, filter PotionFilter) []Potion {
	filtered := make([]Potion, 0)
	for _, list := range potionsByChest {
		for _, p := range list {
			if potionMatchesFilter(p, filter) {
				filtered = append(filtered, p)
			}
		}
	}
	SortPotions(filtered)
	return filtered
}

// ProposedPotions returns the deterministic status:proposed curation queue.
func ProposedPotions(potionsByChest map[string][]Potion) []Potion {
	return FilterPotions(potionsByChest, PotionFilter{Status: domain.PotionStatusProposed})
}

// FindPotion returns the first potion with the given id from grouped potions.
func FindPotion(potionsByChest map[string][]Potion, id string) (Potion, bool) {
	for _, list := range potionsByChest {
		for _, p := range list {
			if p.ID == id {
				return p, true
			}
		}
	}
	return Potion{}, false
}

// SortPotions orders potions by chest id, then potion id.
func SortPotions(potions []Potion) {
	sort.Slice(potions, func(i, k int) bool {
		if potions[i].ChestID != potions[k].ChestID {
			return potions[i].ChestID < potions[k].ChestID
		}
		return potions[i].ID < potions[k].ID
	})
}

func potionMatchesFilter(p Potion, filter PotionFilter) bool {
	if filter.ChestID != "" && p.ChestID != filter.ChestID {
		return false
	}
	switch filter.Status {
	case "":
		return p.Status != domain.PotionStatusDeprecated
	case "all":
		return true
	default:
		return p.Status == filter.Status
	}
}

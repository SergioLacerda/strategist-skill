package treasure

import (
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// JewelFilter selects jewels for read-only listing.
type JewelFilter struct {
	ChestID string
	Status  string
}

// FilterJewels flattens grouped jewels, applies the requested filter, and returns
// a deterministic chest/id ordering for CLI and JSON output.
func FilterJewels(jewelsByChest map[string][]Jewel, filter JewelFilter) []Jewel {
	filtered := make([]Jewel, 0)
	for _, list := range jewelsByChest {
		for _, j := range list {
			if jewelMatchesFilter(j, filter) {
				filtered = append(filtered, j)
			}
		}
	}
	SortJewels(filtered)
	return filtered
}

// ProposedJewels returns the deterministic status:proposed curation queue.
func ProposedJewels(jewelsByChest map[string][]Jewel) []Jewel {
	return FilterJewels(jewelsByChest, JewelFilter{Status: domain.JewelStatusProposed})
}

// FindJewel returns the first jewel with the given id from grouped jewels.
func FindJewel(jewelsByChest map[string][]Jewel, id string) (Jewel, bool) {
	for _, list := range jewelsByChest {
		for _, j := range list {
			if j.ID == id {
				return j, true
			}
		}
	}
	return Jewel{}, false
}

// SortJewels orders jewels by chest id, then jewel id.
func SortJewels(jewels []Jewel) {
	sort.Slice(jewels, func(i, k int) bool {
		if jewels[i].ChestID != jewels[k].ChestID {
			return jewels[i].ChestID < jewels[k].ChestID
		}
		return jewels[i].ID < jewels[k].ID
	})
}

// ParseItemIDs expands comma-separated CLI values into a de-duplicated id list.
// Shared by jewel and potion id parsing — the format has no per-type meaning.
func ParseItemIDs(values ...string) []string {
	ids := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			id := strings.TrimSpace(part)
			if id == "" || seen[id] {
				continue
			}
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

func jewelMatchesFilter(j Jewel, filter JewelFilter) bool {
	if filter.ChestID != "" && j.ChestID != filter.ChestID {
		return false
	}
	switch filter.Status {
	case "":
		return j.Status != domain.JewelStatusDeprecated
	case "all":
		return true
	default:
		return j.Status == filter.Status
	}
}

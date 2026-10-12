package treasuretmp

import (
	"sort"
	"strings"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
)

// Search loads the governed Treasure Chest sources and returns ranked matches.
func Search(req Request) (SearchResponse, error) {
	loaded, err := load(req.Root)
	if err != nil {
		return SearchResponse{}, err
	}
	needles := searchNeedles(req)
	items := append(searchJewels(loaded.jewels, needles), searchPotions(loaded.potions, needles)...)
	sortSearchItems(items)
	items = limitSearchItems(items, req.TokenBudget)
	return SearchResponse{Envelope: envelope(), Items: items}, nil
}

func searchNeedles(req Request) []string {
	needles := make([]string, 0, 1+len(req.Concepts)+len(req.Symbols))
	if strings.TrimSpace(req.Intent) != "" {
		needles = append(needles, req.Intent)
	}
	needles = append(needles, req.Concepts...)
	return append(needles, req.Symbols...)
}

func searchJewels(index map[string][]treasure.Jewel, needles []string) []Item {
	items := make([]Item, 0)
	for _, list := range index {
		for _, jewel := range list {
			if item, ok := searchJewel(jewel, needles); ok {
				items = append(items, item)
			}
		}
	}
	return items
}

func searchJewel(jewel treasure.Jewel, needles []string) (Item, bool) {
	if jewel.Status == "deprecated" {
		return Item{}, false
	}
	score, matched := matchJewel(jewel, needles)
	if !matched {
		return Item{}, false
	}
	return Item{ID: jewel.ID, Kind: "jewel", Score: score, Trust: trustScore(jewel.Trust), Freshness: "fresh", Applicability: jewel.Status, Reason: "jewel metadata matched query", Source: Source{Path: first(jewel.SourceRefs)}}, true
}

func searchPotions(index map[string][]treasure.Potion, needles []string) []Item {
	items := make([]Item, 0)
	for _, list := range index {
		for _, potion := range list {
			if item, ok := searchPotion(potion, needles); ok {
				items = append(items, item)
			}
		}
	}
	return items
}

func searchPotion(potion treasure.Potion, needles []string) (Item, bool) {
	if potion.Status == "deprecated" {
		return Item{}, false
	}
	score, matched := matchPotion(potion, needles)
	if !matched {
		return Item{}, false
	}
	return Item{ID: potion.ID, Kind: "potion", Score: score, Trust: trustScore(potion.Trust), Freshness: "fresh", Applicability: potion.Status, Reason: "runbook metadata matched query", Source: Source{Path: potion.RunbookRef}}, true
}

func sortSearchItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Trust != items[j].Trust {
			return items[i].Trust > items[j].Trust
		}
		return items[i].ID < items[j].ID
	})
}

func limitSearchItems(items []Item, tokenBudget int) []Item {
	limit := tokenBudget
	if limit <= 0 {
		limit = 20
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func matchJewel(jewel treasure.Jewel, needles []string) (float64, bool) {
	if len(needles) == 0 {
		return 1 + float64(jewel.Score.Value)/100, true
	}
	text := strings.Join(append([]string{jewel.ID, jewel.ChestID, jewel.Kind, jewel.Statement}, append(append(jewel.Applicability.Scope, jewel.Applicability.AppliesWhen...), jewel.SourceRefs...)...), " ")
	return matchText(text, needles, float64(jewel.Score.Value)/100)
}

func matchPotion(potion treasure.Potion, needles []string) (float64, bool) {
	if len(needles) == 0 {
		return 1, true
	}
	text := strings.Join([]string{potion.ID, potion.ChestID, potion.WhenToUse, potion.WhenToAvoid, potion.RunbookRef}, " ")
	return matchText(text, needles, 0)
}

func matchText(text string, needles []string, bonus float64) (float64, bool) {
	lower := strings.ToLower(text)
	score := bonus
	matched := false
	for _, needle := range needles {
		needle = strings.TrimSpace(strings.ToLower(needle))
		if needle != "" && strings.Contains(lower, needle) {
			score++
			matched = true
		}
	}
	return score, matched
}

func trustScore(value string) float64 {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "T0":
		return 0.25
	case "T1":
		return 0.50
	case "T2":
		return 0.75
	case "T3":
		return 1.00
	default:
		return 0
	}
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

package leveling

import "strings"

func budgetRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		return 0
	}
}

func evidenceRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal":
		return 1
	case "standard":
		return 2
	case "deep":
		return 3
	default:
		return 0
	}
}

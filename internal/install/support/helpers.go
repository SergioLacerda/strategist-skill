package support

import (
	"sort"
	"strings"
)

// ContainsString reports whether values contains want.
func ContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// SortedSlotNames returns the slot names of slots in ascending order.
func SortedSlotNames(slots map[string]string) []string {
	names := make([]string, 0, len(slots))
	for slot := range slots {
		names = append(names, slot)
	}
	sort.Strings(names)
	return names
}

// PackageID returns the install package id for version, or "dev" when version is empty.
func PackageID(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	return version
}

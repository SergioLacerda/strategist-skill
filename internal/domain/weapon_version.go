package domain

import (
	"strconv"
	"strings"
)

// CompareWeaponVersions orders two Weapon versions numerically by their dotted
// components (so 10.0.0 sorts after 2.0.0), falling back to string order when
// the components tie. It is an ordering for lists and pre-selection only: a
// version is never resolved by being the highest (ADR-0060 Decision 7).
func CompareWeaponVersions(left, right string) int {
	leftParts := weaponVersionParts(left)
	rightParts := weaponVersionParts(right)
	for i := 0; i < len(leftParts) || i < len(rightParts); i++ {
		if comparison := compareWeaponVersionPart(leftParts, rightParts, i); comparison != 0 {
			return comparison
		}
	}
	return strings.Compare(left, right)
}

func compareWeaponVersionPart(left, right []int, index int) int {
	var l, r int
	if index < len(left) {
		l = left[index]
	}
	if index < len(right) {
		r = right[index]
	}
	if l < r {
		return -1
	}
	if l > r {
		return 1
	}
	return 0
}

func weaponVersionParts(version string) []int {
	raw := strings.Split(version, ".")
	parts := make([]int, 0, len(raw))
	for _, part := range raw {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			n = 0
		}
		parts = append(parts, n)
	}
	return parts
}

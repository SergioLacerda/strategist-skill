package domain_test

import (
	"sort"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCompareWeaponVersionsOrdersNumerically(t *testing.T) {
	t.Parallel()

	versions := []string{"2.0.0", "10.0.0", "1.4.0", "1.10.0", "1.9.0"}
	sort.Slice(versions, func(i, j int) bool { return domain.CompareWeaponVersions(versions[i], versions[j]) < 0 })

	assert.Equal(t, []string{"1.4.0", "1.9.0", "1.10.0", "2.0.0", "10.0.0"}, versions, "10 sorts after 2, not between 1 and 2")
	assert.Zero(t, domain.CompareWeaponVersions("1.0.0", "1.0.0"))
	assert.Negative(t, domain.CompareWeaponVersions("1.0", "1.0.0"), "tied components fall back to string order")
}

package rankedbinding

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankedBindingForSlotSelectsByIDAndVersion(t *testing.T) {
	t.Parallel()
	catalog := installtest.RankedVersionsCatalog("1.4.0", "2.0.0")

	binding, err := ForSlot(catalog, "discovery", "demo@2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", binding.WeaponVersion)
	assert.Equal(t, "sha256:b-2.0.0", binding.BindingDigest)

	_, err = ForSlot(catalog, "discovery", "demo")
	require.ErrorContains(t, err, "several certified versions", "a plain id never picks a version")
	require.ErrorContains(t, err, "demo@1.4.0")
	require.ErrorContains(t, err, "demo@2.0.0")

	_, err = ForSlot(catalog, "discovery", "demo@9.9.9")
	require.Error(t, err, "an uncertified version is never offered as Ranked")
}

func TestRankedBindingForSlotAcceptsAPlainIDWhileOnlyOneVersionIsCertified(t *testing.T) {
	t.Parallel()

	binding, err := ForSlot(installtest.RankedVersionsCatalog("1.4.0"), "discovery", "demo")

	require.NoError(t, err)
	assert.Equal(t, "1.4.0", binding.WeaponVersion)
}

func TestEnrichCompiledRankedBindingMigratesAVersionlessLockOnlyWhenUnambiguous(t *testing.T) {
	t.Parallel()
	legacy := domain.SlotBinding{Slot: "discovery", Role: "ranger", InstalledInstanceID: "demo", Mode: domain.SlotBindingModeRanked}

	migrated, ok := EnrichCompiledRankedBinding(installtest.RankedVersionsCatalog("1.4.0"), legacy)
	require.True(t, ok)
	assert.Equal(t, "1.4.0", migrated.WeaponVersion, "the single certified version fills the missing version")
	assert.Equal(t, "sha256:b-1.4.0", migrated.BindingDigest)

	_, ok = EnrichCompiledRankedBinding(installtest.RankedVersionsCatalog("1.4.0", "2.0.0"), legacy)
	assert.False(t, ok, "several certified versions leave a versionless lock unmigrated: no first-match")

	pinned := legacy
	pinned.WeaponVersion = "2.0.0"
	exact, ok := EnrichCompiledRankedBinding(installtest.RankedVersionsCatalog("1.4.0", "2.0.0"), pinned)
	require.True(t, ok)
	assert.Equal(t, "sha256:b-2.0.0", exact.BindingDigest)
}

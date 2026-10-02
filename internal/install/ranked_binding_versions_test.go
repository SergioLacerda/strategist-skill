package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rankedVersionsCatalog(versions ...string) pluginCatalog {
	catalog := pluginCatalog{SchemaVersion: pluginCatalogSchemaVersion}
	for _, version := range versions {
		catalog.RankedBindings = append(catalog.RankedBindings, domain.CompiledRankedBinding{
			Role: "ranger", Slot: "discovery", WeaponID: "demo", WeaponVersion: version,
			WeaponDigest: "sha256:w-" + version, BindingDigest: "sha256:b-" + version, CertificationDigest: "sha256:c-" + version,
			ConnectorID: "strategist-embedded", Entrypoint: "discover", Generation: 1, Status: "active",
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded},
		})
	}
	return catalog
}

func TestRankedBindingForSlotSelectsByIDAndVersion(t *testing.T) {
	t.Parallel()
	catalog := rankedVersionsCatalog("1.4.0", "2.0.0")

	binding, err := rankedBindingForSlot(catalog, "discovery", "demo@2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", binding.WeaponVersion)
	assert.Equal(t, "sha256:b-2.0.0", binding.BindingDigest)

	_, err = rankedBindingForSlot(catalog, "discovery", "demo")
	require.ErrorContains(t, err, "several certified versions", "a plain id never picks a version")
	require.ErrorContains(t, err, "demo@1.4.0")
	require.ErrorContains(t, err, "demo@2.0.0")

	_, err = rankedBindingForSlot(catalog, "discovery", "demo@9.9.9")
	require.Error(t, err, "an uncertified version is never offered as Ranked")
}

func TestRankedBindingForSlotAcceptsAPlainIDWhileOnlyOneVersionIsCertified(t *testing.T) {
	t.Parallel()

	binding, err := rankedBindingForSlot(rankedVersionsCatalog("1.4.0"), "discovery", "demo")

	require.NoError(t, err)
	assert.Equal(t, "1.4.0", binding.WeaponVersion)
}

func TestEnrichCompiledRankedBindingMigratesAVersionlessLockOnlyWhenUnambiguous(t *testing.T) {
	t.Parallel()
	legacy := domain.SlotBinding{Slot: "discovery", Role: "ranger", InstalledInstanceID: "demo", Mode: domain.SlotBindingModeRanked}

	migrated, ok := enrichCompiledRankedBinding(rankedVersionsCatalog("1.4.0"), legacy)
	require.True(t, ok)
	assert.Equal(t, "1.4.0", migrated.WeaponVersion, "the single certified version fills the missing version")
	assert.Equal(t, "sha256:b-1.4.0", migrated.BindingDigest)

	_, ok = enrichCompiledRankedBinding(rankedVersionsCatalog("1.4.0", "2.0.0"), legacy)
	assert.False(t, ok, "several certified versions leave a versionless lock unmigrated: no first-match")

	pinned := legacy
	pinned.WeaponVersion = "2.0.0"
	exact, ok := enrichCompiledRankedBinding(rankedVersionsCatalog("1.4.0", "2.0.0"), pinned)
	require.True(t, ok)
	assert.Equal(t, "sha256:b-2.0.0", exact.BindingDigest)
}

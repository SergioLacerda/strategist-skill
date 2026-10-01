package install

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRankedProviderFailureModes(t *testing.T) {
	t.Parallel()
	binding := domain.SlotBinding{Slot: "refinement", InstalledInstanceID: "openspec-propose"}

	_, _, err := resolveRankedProvider(pluginCatalog{}, binding)
	require.ErrorContains(t, err, "missing from catalog")

	uncertified := pluginCatalog{Providers: []pluginCatalogProvider{{ID: "openspec-propose", Version: "1.0.0"}}}
	_, _, err = resolveRankedProvider(uncertified, binding)
	require.ErrorContains(t, err, "is not certified")

	invalid := pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "openspec-propose", Version: "1.0.0", Ranked: true, CertificationDigest: "sha256:x",
		Runtime: domain.WeaponRuntime{Kind: "bogus"},
	}}}
	_, _, err = resolveRankedProvider(invalid, binding)
	require.ErrorContains(t, err, `ranked runtime provider "openspec-propose"`)
}

func TestBootstrapRankedProviderOnlyActsOnOpenSpecRoot(t *testing.T) {
	t.Parallel()
	private, err := bootstrapRankedProvider(context.Background(), t.TempDir(), pluginCatalogProvider{ID: "p"}, domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded})
	require.NoError(t, err)
	assert.Nil(t, private)

	_, err = bootstrapRankedProvider(context.Background(), t.TempDir(), pluginCatalogProvider{ID: "p"},
		domain.WeaponRuntime{Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/../escape"})
	require.Error(t, err)
}

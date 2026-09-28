package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/require"
)

func rankedNoteCatalog(runtime domain.WeaponRuntime) pluginCatalog {
	return pluginCatalog{Providers: []pluginCatalogProvider{{ID: "openspec-propose", Ranked: true, CertificationDigest: "sha256:c", Runtime: runtime}}}
}

// Choosing Ranked must not be the first time the operator hears that the
// provider runs on the host Node.
func TestRankedRuntimeNoteNamesTheHostNodeMinimum(t *testing.T) {
	catalog := rankedNoteCatalog(domain.WeaponRuntime{Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec"})

	for lang, bundle := range map[string]i18n.WizardStrings{"en": i18n.EN, "pt-BR": i18n.PT} {
		note := rankedRuntimeNote(bundle, catalog, "openspec-propose")

		require.Contains(t, note, "openspec-propose", lang)
		require.Contains(t, note, ">="+domain.MinimumOpenSpecNodeVersion, lang)
	}
}

func TestRankedRuntimeNoteIsEmptyWithoutARuntimeOrRankedOption(t *testing.T) {
	require.Empty(t, rankedRuntimeNote(i18n.EN, rankedNoteCatalog(domain.WeaponRuntime{Kind: domain.RankedRuntimeNone}), "openspec-propose"))
	require.Empty(t, rankedRuntimeNote(i18n.EN, rankedNoteCatalog(domain.WeaponRuntime{Kind: domain.RankedRuntimeOpenSpecRoot}), ""))
	require.Empty(t, rankedRuntimeNote(i18n.EN, pluginCatalog{}, "openspec-propose"))
}

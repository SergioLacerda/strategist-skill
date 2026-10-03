package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestInstallOptionSetRejectsHeadlessRankedChoiceOutsideCertifiedOptions(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	options := newInstallOptionSet(catalog)

	err = options.ValidateRankedSelections(
		map[string]string{"discovery": "not-certified@9.9.9"},
		map[string]string{"discovery": domain.SlotBindingModeRanked},
	)
	require.ErrorContains(t, err, "not certified")
}

func TestInstallOptionSetAllowsCustomChoiceWithoutRankedCertification(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	options := newInstallOptionSet(catalog)

	require.NoError(t, options.ValidateRankedSelections(
		map[string]string{"discovery": "custom-provider"},
		map[string]string{"discovery": domain.SlotBindingModeCustom},
	))
}

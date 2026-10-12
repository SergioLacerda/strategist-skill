package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func writeLegacyCatalog(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	plugins := filepath.Join(root, "plugins")
	require.NoError(t, os.MkdirAll(plugins, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(plugins, "catalog.yaml"), []byte(body), 0o644))
	return root
}

func TestReadinessBlocksLegacyWeaponVocabularyWithoutFallback(t *testing.T) {
	cases := map[string]string{
		"legacy runtime kind": "schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: provider\n    ranked: true\n    certification_digest: sha256:certified\n    runtime:\n      kind: embedded_skill\n",
		"legacy schema":       "schema_version: strategist-plugin-catalog/v1\nproviders:\n  - id: provider\n    ranked: true\n    certification_digest: sha256:certified\n    runtime:\n      kind: embedded\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			root := writeLegacyCatalog(t, body)

			trust, grant, runtimeCheck := rankedCertificationReadiness(root, "discovery", "provider")
			require.Equal(t, domain.ReadinessBlocked, trust.Status)
			require.Equal(t, "ranked_catalog_invalid", trust.ReasonCode)
			require.Contains(t, trust.Detail, "regenerate or reinstall")
			require.Equal(t, trust, grant)
			require.Equal(t, domain.ReadinessUnknown, runtimeCheck.Status)

			custom := customRuntimeReadiness(root, "discovery", "provider")
			require.Equal(t, domain.ReadinessBlocked, custom.Status)
			require.Equal(t, "ranked_catalog_invalid", custom.ReasonCode)
			require.Contains(t, custom.Detail, "regenerate or reinstall")
		})
	}
}

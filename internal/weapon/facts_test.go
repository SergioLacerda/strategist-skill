package weapon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/weapon"
	"github.com/stretchr/testify/require"
)

const factsCatalog = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: demo
    version: "1.0.0"
    risk_score: write_analysis
    canonical_role: ranger
  - id: demo
    version: "2.0.0"
    risk_score: controlled
    canonical_role: sniper
  - id: solo
    risk_score: write_analysis
    canonical_role: archivist
`

func factsRoot(t *testing.T, catalog string) string {
	t.Helper()
	root := t.TempDir()
	if catalog != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(catalog), 0o600))
	}
	return root
}

func writeCustomFacts(t *testing.T, root, instance string) {
	t.Helper()
	lock := "schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: " + instance + "\n    mode: custom\n    status: active\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(lock), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "providers", instance), 0o755))
	adapter := "schema_version: strategist-plugin-adapter/v1\nid: custom\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [refinement]\nsupported_roles: [archivist]\nentrypoints: [host.prompt]\nrisk_score: write_analysis\nscratch_root: runtime\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "providers", instance, "adapter.yaml"), []byte(adapter), 0o600))
}

func TestResolveWeaponFactsUsesCatalogAndVersionRules(t *testing.T) {
	root := factsRoot(t, factsCatalog)

	facts, err := weapon.ResolveWeaponFacts(root, "solo")
	require.NoError(t, err)
	require.Equal(t, domain.WeaponFactsSourceCatalog, facts.Source)
	require.Equal(t, "archivist", facts.CanonicalRole)

	facts, err = weapon.ResolveWeaponFacts(root, "demo@2.0.0")
	require.NoError(t, err)
	require.Equal(t, "controlled", facts.RiskScore)

	_, err = weapon.ResolveWeaponFacts(root, "demo")
	require.ErrorIs(t, err, domain.ErrWeaponFactsAmbiguous)
	_, err = weapon.ResolveWeaponFacts(root, "unknown")
	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)

	all, err := weapon.ListCatalogWeaponFacts(root)
	require.NoError(t, err)
	require.Len(t, all, 3)
	_, found, err := weapon.ResolveCatalogWeaponFacts(root, "solo")
	require.NoError(t, err)
	require.True(t, found)
}

func TestResolveWeaponFactsUsesBoundCustomAdapter(t *testing.T) {
	root := factsRoot(t, "")
	writeCustomFacts(t, root, "custom@1.0.0")

	facts, err := weapon.ResolveWeaponFacts(root, "custom")
	require.NoError(t, err)
	require.Equal(t, domain.WeaponFactsSourceAdapter, facts.Source)
	require.Equal(t, []string{"archivist"}, facts.Roles)

	facts, found, err := weapon.ResolveCustomPackageFacts(root, "custom")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"host.prompt"}, facts.Entrypoints)

	_, found, err = weapon.ResolveCustomPackageFacts(root, "other")
	require.NoError(t, err)
	require.False(t, found)
}

func TestWeaponFactsAdaptersFailClosedOnMissingAndMalformedInputs(t *testing.T) {
	root := factsRoot(t, "")
	_, err := weapon.ResolveWeaponFacts(root, "missing")
	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
	_, err = weapon.ListCatalogWeaponFacts(root)
	require.NoError(t, err)

	require.NoError(t, os.Mkdir(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte("broken"), 0o600))
	_, err = weapon.ResolveWeaponFacts(root, "missing")
	require.ErrorContains(t, err, "parse plugin catalog")

	root = factsRoot(t, "")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("broken"), 0o600))
	_, err = weapon.ResolveWeaponFacts(root, "custom")
	require.ErrorContains(t, err, "parse plugins.lock")

	root = factsRoot(t, "")
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("bindings: []\n"), 0o600))
	_, found, err := weapon.ResolveCustomPackageFacts(root, "custom")
	require.NoError(t, err)
	require.False(t, found)
}

func TestWeaponFactsAdapterReportsMissingAndMalformedAdapter(t *testing.T) {
	root := factsRoot(t, "")
	writeCustomFacts(t, root, "custom@1.0.0")
	path := filepath.Join(root, "providers", "custom@1.0.0", "adapter.yaml")
	require.NoError(t, os.Remove(path))
	_, found, err := weapon.ResolveCustomPackageFacts(root, "custom")
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, os.WriteFile(path, []byte("broken"), 0o600))
	_, found, err = weapon.ResolveCustomPackageFacts(root, "custom")
	require.ErrorContains(t, err, "parse adapter")
	require.False(t, found)
}

package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/require"
)

func TestExternalSkillSourceRosterUsesCanonicalWeaponMetadata(t *testing.T) {
	ids := []string{"brainstorming", "openspec-archive-change", "openspec-explore", "openspec-propose", "writing-plans"}
	for _, id := range ids {
		dir := filepath.Join("..", "..", "external-skills-source", id)
		raw, err := os.ReadFile(filepath.Join(dir, "strategist.yaml")) //nolint:gosec // fixed repository fixture
		require.NoError(t, err, id)
		require.NotContains(t, string(raw), "canonical_role:", id)
		require.NotContains(t, string(raw), "specialization_taxonomy", id)
		adapter, err := loadExternalSkillAdapter(dir, id)
		require.NoError(t, err, id)
		require.NotEmpty(t, adapter.Roles, id)
		require.Contains(t, []domain.WeaponKind{domain.WeaponKindAtomic, domain.WeaponKindComposite}, adapter.Kind, id)
		require.NotEmpty(t, adapter.SupportedSlots, id)
		require.NotEqual(t, domain.RankedRuntimeNone, adapter.Runtime.Kind, id)
	}
}

func TestExternalSkillAdapterRejectsLegacyRuntimeKinds(t *testing.T) {
	for _, legacy := range []string{"embedded_skill", "host_skill"} {
		dir := t.TempDir()
		adapter := "roles:\n  - ranger\nkind: atomic\nsupported_slots:\n  - discovery\nruntime:\n  kind: " + legacy + "\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\nweapon_contract:\n  role_owner: ranger\n  participation: required\n  invocation_evidence: required\n  unavailable_behavior: role_invocation_failed\n  native_substitution: forbidden\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, externalSkillAdapterFileName), []byte(adapter), 0o644))

		_, err := loadExternalSkillAdapter(dir, "legacy")
		require.ErrorContains(t, err, legacy)
		require.ErrorContains(t, err, "regenerate or reinstall")
	}
}

func TestEmbeddedCatalogEntriesAreEmbeddedOriginWeaponsWithCanonicalRuntime(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "plugins", "catalog.yaml")) //nolint:gosec // fixed repository fixture
	require.NoError(t, err)
	catalog, err := parseCatalogBytes(raw)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "embedded_skill")
	require.NotContains(t, string(raw), "host_skill")
	weapons := 0
	for _, provider := range catalog.Providers {
		if provider.CompatibilitySource != "embedded" {
			continue
		}
		weapons++
		require.Equal(t, domain.WeaponOriginEmbedded, provider.Origin, provider.ID)
		require.NoError(t, domain.ValidateActiveRuntimeKind(provider.Runtime.Kind), provider.ID)
	}
	require.NotZero(t, weapons)
}

func TestCustomCatalogProviderIsCustomOriginHostWeapon(t *testing.T) {
	provider := customCatalogProvider(connectors.ResolvedProviderPackage{Package: domain.PluginPackage{ID: "my-weapon", Version: "1.0.0", Digest: "sha256:x"}}, "discovery")

	require.Equal(t, domain.WeaponOriginCustom, provider.Origin)
	require.Equal(t, domain.RankedRuntimeHost, provider.Runtime.Kind)
	require.NoError(t, validateCatalogProvider(provider))
}

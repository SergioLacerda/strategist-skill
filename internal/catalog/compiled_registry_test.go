package catalog_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func compiledRegistryFixture() domain.CompiledRegistry {
	return domain.CompiledRegistry{
		SchemaVersion:   domain.CompiledRegistrySchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Weapons: []domain.CompiledWeapon{{
			ID: "fixture", Version: "1.0.0", Digest: "sha256:weapon",
			Origin:  domain.WeaponOriginEmbedded,
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModeCode},
		}},
		Roles: []domain.CompiledRole{{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role"}},
	}
}

func marshalRegistry(t *testing.T, registry domain.CompiledRegistry) []byte {
	t.Helper()
	raw, err := yaml.Marshal(registry)
	require.NoError(t, err)
	return raw
}

func TestParseCompiledRegistryCatalogDecodesAndValidates(t *testing.T) {
	registry, err := catalog.ParseCompiledRegistryCatalog(marshalRegistry(t, compiledRegistryFixture()))
	require.NoError(t, err)
	weapon, ok := registry.Weapon("fixture", "1.0.0")
	require.True(t, ok)
	require.Equal(t, "sha256:weapon", weapon.Digest)
}

func TestParseCompiledRegistryCatalogReportsDecodeAndValidationErrors(t *testing.T) {
	_, err := catalog.ParseCompiledRegistryCatalog([]byte(":\n - [broken"))
	require.ErrorContains(t, err, "parse compiled registry catalog")

	invalid := compiledRegistryFixture()
	invalid.SchemaVersion = ""
	_, err = catalog.ParseCompiledRegistryCatalog(marshalRegistry(t, invalid))
	require.ErrorContains(t, err, "schema_version is required")
}

func TestCompiledRegistryDriftComparesOnlyDecodedRegistry(t *testing.T) {
	base := marshalRegistry(t, compiledRegistryFixture())
	changed := compiledRegistryFixture()
	changed.Weapons[0].Digest = "sha256:changed"
	different := marshalRegistry(t, changed)

	drift, err := catalog.CompiledRegistryDrift(base, different)
	require.NoError(t, err)
	require.True(t, drift)
	_, err = catalog.CompiledRegistryDrift(base, []byte("broken"))
	require.ErrorContains(t, err, "embedded catalog")
}

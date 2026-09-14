package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCatalogAndMirrorsWritesCatalogMirrorsAndLock(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	writeExternalSkill(t, sourceRoot, "sample-skill", "ranger", "write_analysis", nil)

	result, err := IngestExternalSkills(sourceRoot, pluginCatalog{SchemaVersion: "strategist-plugin-catalog/v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)

	defaultsRoot := t.TempDir()
	catalogPath := filepath.Join(defaultsRoot, "plugins", "catalog.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(catalogPath), 0o755))
	lockPath := filepath.Join(t.TempDir(), "external-skills-source.lock.yaml")

	require.NoError(t, WriteCatalogAndMirrors(result, defaultsRoot, catalogPath, lockPath))

	catalogBytes, err := os.ReadFile(catalogPath)
	require.NoError(t, err)
	assert.Contains(t, string(catalogBytes), "id: sample-skill")

	mirrorPath := filepath.Join(defaultsRoot, "skills", "sample-skill", "skill.yaml")
	mirrorBytes, err := os.ReadFile(mirrorPath)
	require.NoError(t, err)
	assert.Contains(t, string(mirrorBytes), "id: sample-skill")
	assert.Contains(t, string(mirrorBytes), "canonical_role: ranger")

	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(lockBytes), "id: sample-skill")
	assert.Contains(t, string(lockBytes), result.Ingested[0].Package.Digest)
}

func TestWriteCatalogAndMirrorsIsByteIdenticalAcrossReRuns(t *testing.T) {
	t.Parallel()

	sourceRoot := t.TempDir()
	writeExternalSkill(t, sourceRoot, "sample-skill", "ranger", "write_analysis", nil)

	runOnce := func() (catalog, mirror, lock []byte) {
		result, err := IngestExternalSkills(sourceRoot, pluginCatalog{SchemaVersion: "strategist-plugin-catalog/v1"}, domain.TrustPolicy{})
		require.NoError(t, err)

		defaultsRoot := t.TempDir()
		catalogPath := filepath.Join(defaultsRoot, "plugins", "catalog.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(catalogPath), 0o755))
		lockPath := filepath.Join(t.TempDir(), "external-skills-source.lock.yaml")

		require.NoError(t, WriteCatalogAndMirrors(result, defaultsRoot, catalogPath, lockPath))

		c, err := os.ReadFile(catalogPath)
		require.NoError(t, err)
		m, err := os.ReadFile(filepath.Join(defaultsRoot, "skills", "sample-skill", "skill.yaml"))
		require.NoError(t, err)
		l, err := os.ReadFile(lockPath)
		require.NoError(t, err)
		return c, m, l
	}

	c1, m1, l1 := runOnce()
	c2, m2, l2 := runOnce()
	assert.Equal(t, c1, c2)
	assert.Equal(t, m1, m2)
	assert.Equal(t, l1, l2)
}

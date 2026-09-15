package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteCatalogAndMirrors_CatalogWriteErrorPropagates(t *testing.T) {
	t.Parallel()

	result := IngestionResult{Catalog: pluginCatalog{SchemaVersion: "v1"}}
	catalogPath := filepath.Join(t.TempDir(), "no-such-dir", "catalog.yaml")

	err := WriteCatalogAndMirrors(result, t.TempDir(), catalogPath, filepath.Join(t.TempDir(), "lock.yaml"))
	require.ErrorContains(t, err, "write "+catalogPath)
}

func TestWriteCatalogAndMirrors_GenerateMirrorErrorPropagates(t *testing.T) {
	t.Parallel()

	// "ghost" is in Ingested but was never added to Catalog — a hand-built
	// inconsistency real ingestion (buildCatalog) never produces, used here
	// to isolate this one error branch precisely.
	result := IngestionResult{
		Ingested: []IngestedSkill{{ID: "ghost"}},
		Catalog:  pluginCatalog{SchemaVersion: "v1"},
	}
	dir := t.TempDir()

	err := WriteCatalogAndMirrors(result, dir, filepath.Join(dir, "catalog.yaml"), filepath.Join(dir, "lock.yaml"))
	require.ErrorContains(t, err, "generate mirror for ghost")
}

func TestWriteCatalogAndMirrors_MkdirMirrorDirErrorPropagates(t *testing.T) {
	t.Parallel()

	result := IngestionResult{
		Ingested: []IngestedSkill{{ID: "sample"}},
		Catalog:  pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{{ID: "sample", RiskScore: "write_analysis"}}},
	}
	dir := t.TempDir()
	// "skills" exists as a plain file, so MkdirAll(defaultsRoot/skills/sample) fails.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skills"), []byte("x"), 0o644))

	err := WriteCatalogAndMirrors(result, dir, filepath.Join(dir, "catalog.yaml"), filepath.Join(dir, "lock.yaml"))
	require.ErrorContains(t, err, "mkdir")
}

func TestWriteCatalogAndMirrors_WriteMirrorErrorPropagates(t *testing.T) {
	t.Parallel()

	result := IngestionResult{
		Ingested: []IngestedSkill{{ID: "sample"}},
		Catalog:  pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{{ID: "sample", RiskScore: "write_analysis"}}},
	}
	dir := t.TempDir()
	// skill.yaml already exists as a directory, so writing the mirror file fails.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "skills", "sample", "skill.yaml"), 0o755))

	err := WriteCatalogAndMirrors(result, dir, filepath.Join(dir, "catalog.yaml"), filepath.Join(dir, "lock.yaml"))
	require.ErrorContains(t, err, "write "+filepath.Join(dir, "skills", "sample", "skill.yaml"))
}

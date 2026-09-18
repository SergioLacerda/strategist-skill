package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
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

func TestWriteCatalogAndMirrors_FullSuccessWithNestedPackage(t *testing.T) {
	t.Parallel()

	// Source package with a nested subdirectory and a file inside it, to
	// exercise copySkillPackage's walk across both makeSkillDirectory
	// (subdirectory) and copySkillFile (regular file) success paths.
	sourceDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "references", "notes.md"), []byte("notes"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("skill body"), 0o644))

	result := IngestionResult{
		Ingested: []IngestedSkill{{
			ID:      "sample",
			Dir:     sourceDir,
			Package: domain.PluginPackage{Version: "1.0.0", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		}},
		Catalog: pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{{ID: "sample", RiskScore: "write_analysis"}}},
	}
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.yaml")
	lockPath := filepath.Join(dir, "lock.yaml")

	err := WriteCatalogAndMirrors(result, dir, catalogPath, lockPath)
	require.NoError(t, err)

	assert.FileExists(t, catalogPath)
	assert.FileExists(t, lockPath)
	assert.FileExists(t, filepath.Join(dir, "skills", "sample", "skill.yaml"))
	assert.FileExists(t, filepath.Join(dir, "skills", "sample", "SKILL.md"))
	assert.FileExists(t, filepath.Join(dir, "skills", "sample", "references", "notes.md"))

	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(lockBytes), "sample")
	assert.Contains(t, string(lockBytes), "1.0.0")
	assert.Contains(t, string(lockBytes), "original_digest_evidence: verified")
	assert.Contains(t, string(lockBytes), "normalized_digest_evidence: verified")
	assert.NotContains(t, string(lockBytes), "normalized_digest: "+result.Ingested[0].Package.Digest)
}

func TestNormalizedSkillDigestChangesWhenMaterializedOutputChanges(t *testing.T) {
	t.Parallel()

	manifestA := []byte("id: sample\nversion: '1.0.0'\n")
	manifestB := []byte("id: sample\nversion: '2.0.0'\n")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill body"), 0o644))

	digestA, err := normalizedSkillDigest(dir, manifestA)
	require.NoError(t, err)
	digestB, err := normalizedSkillDigest(dir, manifestB)
	require.NoError(t, err)
	assert.NotEqual(t, digestA, digestB)
}

func TestCopySkillPackage_RejectsSymlinks(t *testing.T) {
	t.Parallel()
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "real.md"), []byte("x"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(sourceDir, "real.md"), filepath.Join(sourceDir, "link.md")))

	err := copySkillPackage(sourceDir, t.TempDir())
	require.ErrorContains(t, err, "symlink is not allowed")
}

func TestCopySkillPackage_MissingSourceDirPropagatesWalkError(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	err := copySkillPackage(missing, t.TempDir())
	require.ErrorContains(t, err, "walk "+missing)
}

func TestMakeSkillDirectory_MkdirErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A regular file occupies the path component the directory needs.
	blocker := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := makeSkillDirectory(filepath.Join(blocker, "child"))
	require.ErrorContains(t, err, "mkdir")
}

func TestCopySkillFile_ReadErrorPropagates(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist.md")

	err := copySkillFile(missing, filepath.Join(t.TempDir(), "target.md"))
	require.ErrorContains(t, err, "read "+missing)
}

func TestCopySkillFile_WriteErrorPropagates(t *testing.T) {
	t.Parallel()
	source := filepath.Join(t.TempDir(), "source.md")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	// Target path already exists as a directory, so os.WriteFile must fail.
	target := t.TempDir()

	err := copySkillFile(source, target)
	require.ErrorContains(t, err, "write "+target)
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

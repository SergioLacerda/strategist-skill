package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedDefaultsRootCatalog writes a minimal, valid plugins/catalog.yaml under
// root, the "existing catalog" ingestForOptions/PrepareEmbedded read before
// merging in newly ingested external skills.
func seedDefaultsRootCatalog(t *testing.T, root string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	catalog := "schema_version: v1\nproviders:\n  - id: sniper\n    risk_score: controlled\n    compatibility_source: native_role\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(catalog), 0o644))
}

func TestPrepareEmbedded_WritesCatalogMirrorsAndLock(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeExternalSkill(t, source, "sample-skill", "ranger", "write_analysis", nil)

	defaultsRoot := t.TempDir()
	seedDefaultsRootCatalog(t, defaultsRoot)

	opts := PrepareEmbeddedOptions{
		Source:       source,
		DefaultsRoot: defaultsRoot,
		LockPath:     filepath.Join(t.TempDir(), "lock.yaml"),
	}
	report, err := PrepareEmbedded(opts)
	require.NoError(t, err)
	require.Len(t, report.Ingested, 1)
	assert.Equal(t, "sample-skill", report.Ingested[0].ID)
	assert.Empty(t, report.Rejected)

	catalogBytes, err := os.ReadFile(opts.catalogPath())
	require.NoError(t, err)
	assert.Contains(t, string(catalogBytes), "sample-skill")

	assert.FileExists(t, filepath.Join(defaultsRoot, "skills", "sample-skill", "skill.yaml"))

	lockBytes, err := os.ReadFile(opts.LockPath)
	require.NoError(t, err)
	assert.Contains(t, string(lockBytes), "sample-skill")
}

func TestPrepareEmbedded_IngestErrorPropagates(t *testing.T) {
	t.Parallel()

	defaultsRoot := t.TempDir()
	seedDefaultsRootCatalog(t, defaultsRoot)

	// A Source that is a regular file, not a directory, makes
	// ScanExternalSkillsSourceDirs' os.ReadDir fail with something other than
	// "not exist" (which the scanner otherwise tolerates as "nothing to
	// ingest") — this is what actually reaches IngestExternalSkills' error
	// return.
	sourceFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(sourceFile, []byte("x"), 0o644))

	opts := PrepareEmbeddedOptions{
		Source:       sourceFile,
		DefaultsRoot: defaultsRoot,
		LockPath:     filepath.Join(t.TempDir(), "lock.yaml"),
	}
	_, err := PrepareEmbedded(opts)
	require.Error(t, err)
	require.ErrorContains(t, err, "ingest external skills")
}

func TestPrepareEmbedded_WriteErrorPropagates(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	writeExternalSkill(t, source, "sample-skill", "ranger", "write_analysis", nil)

	defaultsRoot := t.TempDir()
	seedDefaultsRootCatalog(t, defaultsRoot)

	// Ingestion succeeds (valid Source, valid existing catalog), but
	// WriteCatalogAndMirrors's final os.WriteFile(lockPath, ...) fails:
	// LockPath's parent directory does not exist and WriteFile never creates
	// intermediate directories.
	opts := PrepareEmbeddedOptions{
		Source:       source,
		DefaultsRoot: defaultsRoot,
		LockPath:     filepath.Join(t.TempDir(), "no-such-subdir", "lock.yaml"),
	}
	_, err := PrepareEmbedded(opts)
	require.Error(t, err)
	require.ErrorContains(t, err, "lock.yaml")
}

func TestIngestForOptions_UnreadableCatalog(t *testing.T) {
	t.Parallel()

	opts := PrepareEmbeddedOptions{
		Source:       t.TempDir(),
		DefaultsRoot: filepath.Join(t.TempDir(), "does-not-exist"),
		LockPath:     filepath.Join(t.TempDir(), "lock.yaml"),
	}
	_, err := ingestForOptions(opts)
	require.Error(t, err)
	require.ErrorContains(t, err, "read")
}

func TestIngestForOptions_InvalidCatalog(t *testing.T) {
	t.Parallel()

	defaultsRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(defaultsRoot, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(defaultsRoot, "plugins", "catalog.yaml"), []byte("schema_version: [unterminated"), 0o644))

	opts := PrepareEmbeddedOptions{
		Source:       t.TempDir(),
		DefaultsRoot: defaultsRoot,
		LockPath:     filepath.Join(t.TempDir(), "lock.yaml"),
	}
	_, err := ingestForOptions(opts)
	require.Error(t, err)
	require.ErrorContains(t, err, "parse")
}

func TestCatalogHasDrifted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.yaml")
	want := filepath.Join(dir, "want.yaml")

	t.Run("existing unreadable", func(t *testing.T) {
		_, err := catalogHasDrifted(filepath.Join(dir, "missing.yaml"), want)
		require.Error(t, err)
	})

	require.NoError(t, os.WriteFile(existing, []byte("a"), 0o644))

	t.Run("want unreadable", func(t *testing.T) {
		_, err := catalogHasDrifted(existing, filepath.Join(dir, "missing-want.yaml"))
		require.Error(t, err)
	})

	require.NoError(t, os.WriteFile(want, []byte("b"), 0o644))
	t.Run("differs", func(t *testing.T) {
		drifted, err := catalogHasDrifted(existing, want)
		require.NoError(t, err)
		assert.True(t, drifted)
	})

	require.NoError(t, os.WriteFile(want, []byte("a"), 0o644))
	t.Run("same", func(t *testing.T) {
		drifted, err := catalogHasDrifted(existing, want)
		require.NoError(t, err)
		assert.False(t, drifted)
	})
}

func TestMirrorsHaveDrifted(t *testing.T) {
	t.Parallel()

	t.Run("no ingested skills", func(t *testing.T) {
		drifted, err := mirrorsHaveDrifted(nil, t.TempDir(), t.TempDir())
		require.NoError(t, err)
		assert.False(t, drifted)
	})

	t.Run("want file missing errors", func(t *testing.T) {
		_, err := mirrorsHaveDrifted([]IngestedSkill{{ID: "sample"}}, t.TempDir(), t.TempDir())
		require.Error(t, err)
	})

	t.Run("got file missing counts as drifted", func(t *testing.T) {
		tmpRoot := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "skills", "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "skills", "sample", "skill.yaml"), []byte("a"), 0o644))

		drifted, err := mirrorsHaveDrifted([]IngestedSkill{{ID: "sample"}}, t.TempDir(), tmpRoot)
		require.NoError(t, err)
		assert.True(t, drifted)
	})

	t.Run("differing content counts as drifted", func(t *testing.T) {
		tmpRoot := t.TempDir()
		defaultsRoot := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "skills", "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "skills", "sample", "skill.yaml"), []byte("a"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(defaultsRoot, "skills", "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(defaultsRoot, "skills", "sample", "skill.yaml"), []byte("b"), 0o644))

		drifted, err := mirrorsHaveDrifted([]IngestedSkill{{ID: "sample"}}, defaultsRoot, tmpRoot)
		require.NoError(t, err)
		assert.True(t, drifted)
	})

	t.Run("identical content is not drifted", func(t *testing.T) {
		tmpRoot := t.TempDir()
		defaultsRoot := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "skills", "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "skills", "sample", "skill.yaml"), []byte("same"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(defaultsRoot, "skills", "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(defaultsRoot, "skills", "sample", "skill.yaml"), []byte("same"), 0o644))

		drifted, err := mirrorsHaveDrifted([]IngestedSkill{{ID: "sample"}}, defaultsRoot, tmpRoot)
		require.NoError(t, err)
		assert.False(t, drifted)
	})
}

func TestLockHasDrifted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missingLock := filepath.Join(dir, "missing-lock.yaml")
	want := filepath.Join(dir, "want-lock.yaml")
	require.NoError(t, os.WriteFile(want, []byte("content"), 0o644))

	t.Run("missing existing lock with nothing ingested is not drift", func(t *testing.T) {
		drifted, err := lockHasDrifted(missingLock, want, false)
		require.NoError(t, err)
		assert.False(t, drifted)
	})

	t.Run("missing existing lock with something ingested is drift", func(t *testing.T) {
		drifted, err := lockHasDrifted(missingLock, want, true)
		require.NoError(t, err)
		assert.True(t, drifted)
	})

	existing := filepath.Join(dir, "existing-lock.yaml")
	require.NoError(t, os.WriteFile(existing, []byte("content"), 0o644))

	t.Run("want unreadable errors", func(t *testing.T) {
		_, err := lockHasDrifted(existing, filepath.Join(dir, "missing-want.yaml"), true)
		require.Error(t, err)
	})

	t.Run("same content is not drift", func(t *testing.T) {
		drifted, err := lockHasDrifted(existing, want, true)
		require.NoError(t, err)
		assert.False(t, drifted)
	})

	require.NoError(t, os.WriteFile(want, []byte("different"), 0o644))
	t.Run("different content is drift", func(t *testing.T) {
		drifted, err := lockHasDrifted(existing, want, true)
		require.NoError(t, err)
		assert.True(t, drifted)
	})
}

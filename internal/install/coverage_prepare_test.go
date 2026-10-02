package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestForOptionsFailureModes(t *testing.T) {
	t.Parallel()
	source := t.TempDir()

	missing := PrepareEmbeddedOptions{Source: source, DefaultsRoot: t.TempDir(), LockPath: filepath.Join(t.TempDir(), "lock.yaml")}
	_, err := PrepareEmbedded(missing)
	require.ErrorContains(t, err, "read ")
	_, _, err = CheckEmbeddedDrift(missing)
	require.Error(t, err)

	malformed := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(malformed, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(malformed, "plugins", "catalog.yaml"), []byte("providers: [unclosed"), 0o644))
	_, err = PrepareEmbedded(PrepareEmbeddedOptions{Source: source, DefaultsRoot: malformed})
	require.ErrorContains(t, err, "parse ")

	noRoles := t.TempDir()
	seedDefaultsRootCatalog(t, noRoles)
	require.NoError(t, os.Remove(filepath.Join(noRoles, "roles", "default.yaml")))
	_, err = PrepareEmbedded(PrepareEmbeddedOptions{Source: source, DefaultsRoot: noRoles})
	require.ErrorContains(t, err, "build compiled registry")
}

func TestCheckEmbeddedDriftReportsCleanAndDriftedState(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	writeExternalSkill(t, source, "sample-skill", "ranger", "write_analysis", nil)
	defaultsRoot := t.TempDir()
	seedDefaultsRootCatalog(t, defaultsRoot)
	opts := PrepareEmbeddedOptions{Source: source, DefaultsRoot: defaultsRoot, LockPath: filepath.Join(t.TempDir(), "lock.yaml")}

	_, drift, err := CheckEmbeddedDrift(opts)
	require.NoError(t, err)
	assert.True(t, drift, "nothing was prepared yet")

	_, err = PrepareEmbedded(opts)
	require.NoError(t, err)
	_, drift, err = CheckEmbeddedDrift(opts)
	require.NoError(t, err)
	assert.False(t, drift)

	require.NoError(t, os.WriteFile(filepath.Join(defaultsRoot, "skills", "sample-skill@1.0.0", "SKILL.md"), []byte("tampered"), 0o644))
	_, drift, err = CheckEmbeddedDrift(opts)
	require.NoError(t, err)
	assert.True(t, drift, "a changed mirror is drift")

	require.NoError(t, os.Remove(opts.LockPath))
	_, drift, err = CheckEmbeddedDrift(opts)
	require.NoError(t, err)
	assert.True(t, drift, "a missing lock is drift")
}

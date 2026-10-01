package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubHostNode(t *testing.T, find func(string) (string, error), version func(context.Context, string, string) ([]byte, error)) {
	t.Helper()
	origFind, origVersion := findHostNode, hostNodeVersion
	findHostNode, hostNodeVersion = find, version
	t.Cleanup(func() { findHostNode, hostNodeVersion = origFind, origVersion })
}

func TestResolveHostNode(t *testing.T) {
	ctx := context.Background()
	stubHostNode(t, func(string) (string, error) { return "", errors.New("not on PATH") }, nil)
	_, err := resolveHostNode(ctx, t.TempDir())
	require.Error(t, err)

	stubHostNode(t, func(string) (string, error) { return "node", nil },
		func(context.Context, string, string) ([]byte, error) { return nil, errors.New("exec failed") })
	_, err = resolveHostNode(ctx, t.TempDir())
	require.ErrorContains(t, err, "version")

	stubHostNode(t, func(string) (string, error) { return "node", nil },
		func(context.Context, string, string) ([]byte, error) { return []byte("v1.0.0\n"), nil })
	_, err = resolveHostNode(ctx, t.TempDir())
	require.ErrorContains(t, err, "is required to run the embedded OpenSpec bundle")

	stubHostNode(t, func(string) (string, error) { return "node", nil },
		func(context.Context, string, string) ([]byte, error) { return []byte("v99.0.0\n"), nil })
	abs, err := resolveHostNode(ctx, t.TempDir())
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(abs))

	_, _, err = resolveRankedExecutable(ctx, t.TempDir(), "openspec-propose", "1.0.0", domain.WeaponRuntime{})
	require.ErrorContains(t, err, "embedded OpenSpec runtime", "an unknown bundle version cannot be materialized")
}

func TestCheckOpenSpecPinAndRelSlash(t *testing.T) {
	t.Parallel()
	state := &domain.RankedRuntimeStateRuntime{Components: []domain.RankedRuntimeStateComponent{{Name: "openspec", Version: "1.13.0"}, {Name: "other", Version: "9"}}}
	require.NoError(t, checkOpenSpecPin(domain.WeaponRuntime{}, state), "unpinned contract accepts any bundle")
	require.NoError(t, checkOpenSpecPin(domain.WeaponRuntime{Version: "1.13.0"}, state))
	require.ErrorContains(t, checkOpenSpecPin(domain.WeaponRuntime{Version: "2.0.0"}, state), "ranked_runtime_pin_mismatch")

	assert.Equal(t, "a/b", relSlash("/base", "/base/a/b"))
	assert.Equal(t, "rel", relSlash("/base", "rel"), "an unrelatable target is returned as is")
}

func TestRemoveLegacyNestedOpenSpecRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, removeLegacyNestedOpenSpecRoot(root), "no legacy root")

	legacy := filepath.Join(root, "openspec")
	require.NoError(t, os.MkdirAll(filepath.Join(legacy, "changes", "archive"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(legacy, "specs"), 0o755))
	for _, name := range []string{"config.yaml", "changes/archive/.gitkeep", "specs/.gitkeep"} {
		require.NoError(t, os.WriteFile(filepath.Join(legacy, filepath.FromSlash(name)), nil, 0o644))
	}
	require.NoError(t, removeLegacyNestedOpenSpecRoot(root))
	assert.NoDirExists(t, legacy)

	require.NoError(t, os.MkdirAll(filepath.Join(legacy, "specs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "config.yaml"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "specs", "mine.md"), []byte("user"), 0o644))
	require.ErrorContains(t, removeLegacyNestedOpenSpecRoot(root), "contains user content")
	assert.FileExists(t, filepath.Join(legacy, "specs", "mine.md"), "user content is never deleted")

	_, err := findUnexpectedLegacyPath(filepath.Join(root, "absent"), legacyOpenSpecAllowedPaths())
	require.ErrorContains(t, err, "walk legacy OpenSpec root")
}

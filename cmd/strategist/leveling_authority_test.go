package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestLoadLevelingPolicyUsesSharedAuthority(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o700))
	defaults, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(defaults)
	require.NoError(t, err)
	manifest, err := json.Marshal(domain.InstallManifest{
		Schema:                "strategist.install-manifest.v1",
		LevelingPolicyVersion: policy.Version,
		LevelingPolicyDigest:  policy.Digest(),
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, domain.InstallManifestRelPath), manifest, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "leveling.yaml"), []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n"), 0o600))

	project := filepath.Dir(root)
	oldCWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(project))
	t.Cleanup(func() { _ = os.Chdir(oldCWD) })

	got, _, err := loadLevelingPolicy()
	require.NoError(t, err)
	require.Equal(t, "medium", got.Defaults.Roles["ranger"].Effort)
	require.NotEmpty(t, got.Providers["CODEX"].Models)
}

func TestLoadLevelingPolicyFailsClosedForUnreadableManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o700))
	defaults, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "leveling.yaml"), defaults, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(root, domain.InstallManifestRelPath), 0o700))

	project := filepath.Dir(root)
	oldCWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(project))
	t.Cleanup(func() { _ = os.Chdir(oldCWD) })

	_, _, err = loadLevelingPolicy()
	require.ErrorContains(t, err, "leveling_policy_stale")
}

func TestLoadLevelingPolicyAcceptsDocumentedLegacyMarkerWithoutOverrideFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 1\nmode: legacy\n"), 0o600))

	project := filepath.Dir(root)
	oldCWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(project))
	t.Cleanup(func() { _ = os.Chdir(oldCWD) })

	got, _, err := loadLevelingPolicy()
	require.NoError(t, err)
	require.NotEmpty(t, got.Providers["CODEX"].Models)
}

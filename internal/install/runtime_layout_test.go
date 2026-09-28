package install

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Install and upgrade both stamp the binary's layout generation into the manifest,
// and a manifest written before the marker existed reads as generation 0.
func TestInstallStampsTheRuntimeLayoutGeneration(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, runtimeDefaultService(newRuntimeDefaultsExtractor(nil)).Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true, NoShim: true}))

	manifest, found, err := loadInstallManifest(filepath.Join(dir, ".strategist"))
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, domain.RuntimeLayoutGeneration, manifest.RuntimeLayoutGeneration)
}

func TestApplyUpgradeStampsTheRuntimeLayoutGeneration(t *testing.T) {
	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)

	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	manifest, found, err := loadInstallManifest(dir)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, domain.RuntimeLayoutGeneration, manifest.RuntimeLayoutGeneration)
}

package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writtenActiveConfig(t *testing.T, wc domain.WizardConfig) (string, domain.ActiveConfig) {
	t.Helper()
	dir := t.TempDir()
	wc.Mode, wc.BasePath = "epic", ".analysis"
	wc.DiscoveryProvider, wc.RefinementProvider, wc.ExecutionProvider = "brainstorming", "openspec-propose", "sniper"
	require.NoError(t, writeActiveYAML(dir, wc))
	raw, err := os.ReadFile(filepath.Join(dir, activeYAMLName))
	require.NoError(t, err)
	var cfg domain.ActiveConfig
	require.NoError(t, yaml.Unmarshal(raw, &cfg))
	return string(raw), cfg
}

func TestWriteActiveYAMLOmitsLevelingWhenNotAnswered(t *testing.T) {
	raw, cfg := writtenActiveConfig(t, domain.WizardConfig{})
	assert.NotContains(t, raw, "leveling:")
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.Leveling.EffectiveMode())
	require.NoError(t, cfg.Validate())
}

func TestWriteActiveYAMLRecordsAutomaticChoice(t *testing.T) {
	raw, cfg := writtenActiveConfig(t, domain.WizardConfig{Leveling: domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}})
	assert.Contains(t, raw, "leveling:\n  mode: automatic\n")
	assert.Contains(t, raw, "automatic = the LEVELING policy picks them per")
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.Leveling.EffectiveMode())
}

func TestWriteActiveYAMLRecordsManualModeOnly(t *testing.T) {
	raw, cfg := writtenActiveConfig(t, domain.WizardConfig{Leveling: domain.LevelingConfig{Mode: domain.LevelingModeManual}})
	require.NoError(t, cfg.Validate())
	assert.True(t, cfg.Leveling.HostPassthrough())
	assert.Contains(t, raw, "manual = the host's model and effort are used as-is")
	assert.NotContains(t, raw, "roles:", "manual stores no per-role values")
}

func TestLevelingActiveYAMLRejectsUnknownMode(t *testing.T) {
	_, err := levelingActiveYAML(domain.LevelingConfig{Mode: "smart"})
	require.ErrorContains(t, err, "leveling_mapping_invalid")
}

func TestApplyUpgradePreservesCustomLevelingBlock(t *testing.T) {
	dir := t.TempDir()
	original := "mode: epic\nbase_path: .analysis\nleveling:\n  mode: manual\n"
	activePath := filepath.Join(dir, activeYAMLName)
	require.NoError(t, os.WriteFile(activePath, []byte(original), 0o600))

	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	got, err := os.ReadFile(activePath) //nolint:gosec // test temp file
	require.NoError(t, err)
	assert.Equal(t, original, string(got), "upgrade never rewrites the operator's active.yaml")
}

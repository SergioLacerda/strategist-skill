package install

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestValidateWizardLevelingRejectsMissingPolicy(t *testing.T) {
	require.ErrorContains(t, validateWizardLeveling(t.TempDir(), domain.WizardConfig{DiscoveryProvider: "CODEX"}), "leveling_policy_missing")
}

func TestValidateWizardLevelingAllowsExplicitLegacyCompatibility(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, levelingCompatibilityPath), []byte("version: 1\nmode: legacy\n"), 0o600))
	require.NoError(t, validateWizardLeveling(dir, domain.WizardConfig{DiscoveryProvider: "new-ranked"}))
}

func TestValidateWizardLevelingChecksSelectedBindings(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "leveling.yaml"), raw, 0o600))
	require.NoError(t, validateWizardLeveling(dir, domain.WizardConfig{
		DiscoveryProvider: "CODEX", RefinementProvider: "CLAUDE", ExecutionProvider: "new-ranked",
	}))
}

func TestValidateWizardLevelingRejectsInvalidPolicy(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "leveling.yaml"), []byte("version: 1\n"), 0o600))
	require.ErrorContains(t, validateWizardLeveling(dir, domain.WizardConfig{DiscoveryProvider: "CODEX"}), "load LEVELING policy")
}

func TestValidateWizardLevelingRejectsStaleInstallAuthority(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, levelingPolicyPath), raw, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, domain.InstallManifestRelPath), []byte(`{"schema":"strategist.install-manifest.v1","leveling_policy_version":1,"leveling_policy_digest":"stale"}`), 0o600))
	require.ErrorContains(t, validateWizardLeveling(dir, domain.WizardConfig{DiscoveryProvider: "CODEX"}), "leveling_policy_stale")
}

func TestValidateWizardLevelingAcceptsPartialOverride(t *testing.T) {
	dir := t.TempDir()
	defaults, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, levelingPolicyPath), []byte("version: 1\ndefaults:\n  roles:\n    ranger:\n      effort: medium\n"), 0o600))
	policy, err := leveling.Parse(defaults)
	require.NoError(t, err)
	manifest := fmt.Sprintf(`{"schema":"strategist.install-manifest.v1","leveling_policy_version":%d,"leveling_policy_digest":"%s"}`, policy.Version, policy.Digest())
	require.NoError(t, os.WriteFile(filepath.Join(dir, domain.InstallManifestRelPath), []byte(manifest), 0o600))
	require.NoError(t, validateWizardLeveling(dir, domain.WizardConfig{DiscoveryProvider: "CODEX"}, minimalExtractor{}))
}

func TestValidateWizardLevelingUsesPortableRuntimeRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Windows Client", "repo", ".strategist")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, levelingPolicyPath), raw, 0o600))
	require.NoError(t, validateWizardLeveling(dir, domain.WizardConfig{DiscoveryProvider: "CODEX"}, minimalExtractor{}))
}

func manualLevelingFor(roles ...string) domain.LevelingConfig {
	choices := map[string]domain.LevelingRoleChoice{}
	for _, role := range roles {
		choices[role] = domain.LevelingRoleChoice{Model: "Sonnet", Effort: "high"}
	}
	return domain.LevelingConfig{Mode: domain.LevelingModeManual, Roles: choices}
}

func TestValidateWizardLevelingSkipsPolicyWhenEverySlotRoleIsManual(t *testing.T) {
	wc := domain.WizardConfig{DiscoveryProvider: "CODEX", RefinementProvider: "CLAUDE", ExecutionProvider: "sniper", Leveling: manualLevelingFor("ranger", "archivist", "sniper")}
	// No leveling.yaml and no compatibility marker: reading the policy would fail.
	require.NoError(t, validateWizardLeveling(t.TempDir(), wc))
}

func TestValidateWizardLevelingStillChecksAutomaticRoles(t *testing.T) {
	wc := domain.WizardConfig{DiscoveryProvider: "CODEX", RefinementProvider: "CLAUDE", Leveling: manualLevelingFor("ranger")}
	require.ErrorContains(t, validateWizardLeveling(t.TempDir(), wc), "leveling_policy_missing")
}

func TestValidateWizardLevelingIgnoresIncompleteManualChoice(t *testing.T) {
	wc := domain.WizardConfig{DiscoveryProvider: "CODEX", Leveling: domain.LevelingConfig{Mode: domain.LevelingModeManual, Roles: map[string]domain.LevelingRoleChoice{"ranger": {Effort: "high"}}}}
	require.ErrorContains(t, validateWizardLeveling(t.TempDir(), wc), "leveling_policy_missing", "a partial choice still needs the policy to complete it")
}

func TestValidateWizardLevelingAutomaticModeUnchanged(t *testing.T) {
	wc := domain.WizardConfig{DiscoveryProvider: "CODEX", Leveling: domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}}
	require.ErrorContains(t, validateWizardLeveling(t.TempDir(), wc), "leveling_policy_missing")
}

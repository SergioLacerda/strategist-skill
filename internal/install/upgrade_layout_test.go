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

// legacyInstall lays down a workspace as a pre-ADR-0061 install left it: the
// id-only skills/demo/ layout, tracked by an install manifest.
func legacyInstall(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	hashes := make(map[string]string, len(files))
	for rel, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		hashes[rel] = domain.SHA256Hex([]byte(body))
	}
	require.NoError(t, saveInstallManifest(dir, domain.NewFullInstallManifest("pre-0061", hashes)))
	return dir
}

func stateOf(plan UpgradePlan, path string) domain.RuntimeFileUpgradeState {
	for _, entry := range plan.Entries {
		if entry.Path == path {
			return entry.State
		}
	}
	return ""
}

var versionedDefaults = map[string][]byte{
	"skills/demo@1.0.0/SKILL.md":          []byte("payload"),
	"skills/demo@1.0.0/skill.yaml":        []byte("id: demo\n"),
	"skills/demo@1.0.0/runtime/bundle.js": []byte("bundle"),
}

func legacyDemoFiles() map[string]string {
	return map[string]string{
		"skills/demo/SKILL.md":          "payload",
		"skills/demo/skill.yaml":        "id: demo\n",
		"skills/demo/runtime/bundle.js": "bundle",
		"old/unrelated.md":              "was shipped once",
	}
}

func TestPlanUpgradeClassifiesTheLegacyIDOnlyLayoutAsMigratable(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, legacyDemoFiles())

	plan, err := upgradeTestService(versionedDefaults).PlanUpgrade(dir)
	require.NoError(t, err)

	assert.Equal(t, domain.UpgradeLegacyLayout, stateOf(plan, "skills/demo/SKILL.md"))
	assert.Equal(t, domain.UpgradeLegacyLayout, stateOf(plan, "skills/demo/runtime/bundle.js"))
	assert.Equal(t, domain.UpgradeOrphaned, stateOf(plan, "skills/demo/skill.yaml"), "the id-only compat view is owned by the installer, never migrated")
	assert.Equal(t, domain.UpgradeOrphaned, stateOf(plan, "old/unrelated.md"), "an orphan outside the skills layout is only reported")
}

func TestPlanUpgradeKeepsAModifiedLegacyFileAsAnOrphan(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, legacyDemoFiles())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skills", "demo", "SKILL.md"), []byte("edited by the operator"), 0o644))

	plan, err := upgradeTestService(versionedDefaults).PlanUpgrade(dir)
	require.NoError(t, err)

	assert.Equal(t, domain.UpgradeOrphaned, stateOf(plan, "skills/demo/SKILL.md"), "a customized file is never deleted")
	assert.Equal(t, domain.UpgradeLegacyLayout, stateOf(plan, "skills/demo/runtime/bundle.js"))
}

func TestPlanUpgradeLeavesAnIDWithNoVersionedCounterpartAlone(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, map[string]string{"skills/other/SKILL.md": "x"})

	plan, err := upgradeTestService(versionedDefaults).PlanUpgrade(dir)
	require.NoError(t, err)

	assert.Equal(t, domain.UpgradeOrphaned, stateOf(plan, "skills/other/SKILL.md"), "only ids the new layout replaces are migrated")
}

func TestApplyUpgradeMigratesTheLegacyLayoutAndRollbackRestoresIt(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, legacyDemoFiles())
	svc := upgradeTestService(versionedDefaults)
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)

	backup, err := svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)
	require.NotEmpty(t, backup, "files about to be removed are snapshotted first")

	assert.FileExists(t, filepath.Join(dir, "skills", "demo@1.0.0", "SKILL.md"), "the versioned layout is written")
	assert.NoFileExists(t, filepath.Join(dir, "skills", "demo", "SKILL.md"))
	assert.NoFileExists(t, filepath.Join(dir, "skills", "demo", "runtime", "bundle.js"))
	assert.NoDirExists(t, filepath.Join(dir, "skills", "demo", "runtime"), "emptied directories are pruned")
	assert.FileExists(t, filepath.Join(dir, "skills", "demo", "skill.yaml"), "the compat view stays")
	assert.FileExists(t, filepath.Join(dir, "old", "unrelated.md"))

	stamp := filepath.Base(backup)
	restored, err := RollbackUpgrade(dir, stamp)
	require.NoError(t, err)
	assert.Positive(t, restored)
	body, err := os.ReadFile(filepath.Join(dir, "skills", "demo", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, "payload", string(body), "rollback brings the legacy files back")
	assert.FileExists(t, filepath.Join(dir, "skills", "demo", "runtime", "bundle.js"))
}

// upgradeCatalogFile is a minimal catalog with one certified Ranked binding.
func upgradeCatalogFile(t *testing.T) []byte {
	t.Helper()
	catalog := rankedVersionsCatalog("1.0.0")
	catalog.Providers = []pluginCatalogProvider{{
		ID: "demo", Version: "1.0.0", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"},
		CompatibilitySource: "embedded", Ranked: true, CertificationDigest: "sha256:c-1.0.0",
	}}
	raw, err := yaml.Marshal(catalog)
	require.NoError(t, err)
	return raw
}

func legacyRankedLock(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, writePluginLockFile(dir, domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		Slot: "discovery", Role: "ranger", InstalledInstanceID: "demo", Mode: domain.SlotBindingModeRanked, Status: "enabled",
	}}}))
}

func TestApplyUpgradeMigratesAVersionlessRankedLockAndRollbackRestoresIt(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, map[string]string{"README.md": "x"})
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	legacyRankedLock(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugins", "catalog.yaml"), upgradeCatalogFile(t), 0o644))
	before, err := os.ReadFile(filepath.Join(dir, pluginLockFileName))
	require.NoError(t, err)
	defaults := map[string][]byte{"plugins/catalog.yaml": upgradeCatalogFile(t), "README.md": []byte("x")}
	svc := upgradeTestService(defaults)
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"discovery"}, plan.LockMigration, "the plan names the slots whose lock gains a weapon_version")

	backup, err := svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	migrated, err := readPluginLockFile(dir)
	require.NoError(t, err)
	require.Len(t, migrated.Bindings, 1)
	assert.Equal(t, "1.0.0", migrated.Bindings[0].WeaponVersion, "the single certified version fills the missing one")
	assert.Equal(t, "sha256:b-1.0.0", migrated.Bindings[0].BindingDigest)

	require.NotEmpty(t, backup, "the lock is snapshotted before it changes")
	_, err = RollbackUpgrade(dir, filepath.Base(backup))
	require.NoError(t, err)
	after, err := os.ReadFile(filepath.Join(dir, pluginLockFileName))
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "rollback restores the lock byte for byte")
}

func TestPlanUpgradeReportsNoLockMigrationForAnAlreadyVersionedLock(t *testing.T) {
	t.Parallel()
	dir := legacyInstall(t, map[string]string{"README.md": "x"})
	require.NoError(t, writePluginLockFile(dir, domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		Slot: "discovery", InstalledInstanceID: "demo", WeaponVersion: "1.0.0", Mode: domain.SlotBindingModeRanked,
	}}}))

	plan, err := upgradeTestService(map[string][]byte{"README.md": []byte("x")}).PlanUpgrade(dir)

	require.NoError(t, err)
	assert.Empty(t, plan.LockMigration)
}

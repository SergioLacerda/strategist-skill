package install

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internalinstall "github.com/SergioLacerda/strategist-skill/internal/install"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUpgrader struct {
	plan          internalinstall.UpgradePlan
	planErr       error
	applyErr      error
	backupDir     string
	planCalls     int
	applyCalls    int
	gotDir        string
	gotForce      bool
	factoryDowngr *bool
}

func (f *fakeUpgrader) PlanUpgrade(dir string) (internalinstall.UpgradePlan, error) {
	f.planCalls++
	f.gotDir = dir
	return f.plan, f.planErr
}

func (f *fakeUpgrader) ApplyUpgrade(_ string, _ internalinstall.UpgradePlan, force bool) (string, error) {
	f.applyCalls++
	f.gotForce = force
	return f.backupDir, f.applyErr
}

func fakeUpgradeDeps(f *fakeUpgrader, target string) UpgradeDependencies {
	return UpgradeDependencies{
		ResolveTarget: func(explicit string, _ bool) (string, error) {
			if explicit != "" {
				return explicit, nil
			}
			return target, nil
		},
		ServiceFactory: func(allowDowngrade bool) UpgradeService {
			f.factoryDowngr = &allowDowngrade
			return f
		},
	}
}

func upgradeOutput(t *testing.T, deps UpgradeDependencies, flags map[string]string) (string, error) {
	t.Helper()
	cmd := NewUpgrade(deps)
	for name, value := range flags {
		require.NoError(t, cmd.Flags().Set(name, value), name)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := cmd.RunE(cmd, nil)
	return out.String(), err
}

func samplePlan() internalinstall.UpgradePlan {
	return internalinstall.UpgradePlan{Entries: []internalinstall.UpgradePlanEntry{
		{Path: "a.md", State: domain.UpgradeManaged},
		{Path: "b.md", State: domain.UpgradeMissing},
		{Path: "c.md", State: domain.UpgradeAutoUpgrade},
		{Path: "d.md", State: domain.UpgradeCustomized},
		{Path: "e.md", State: domain.UpgradeOrphaned},
	}}
}

func TestNewUpgrade_KeepsCommandContract(t *testing.T) {
	cmd := NewUpgrade(UpgradeDependencies{})

	assert.Equal(t, "upgrade", cmd.Use)
	for _, name := range []string{"target", "global", "dry-run", "force", "allow-downgrade", "rollback"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), name)
	}
}

func TestUpgrade_DryRunPrintsPlanAndWritesNothing(t *testing.T) {
	fake := &fakeUpgrader{plan: samplePlan()}

	out, err := upgradeOutput(t, fakeUpgradeDeps(fake, ""), map[string]string{"target": "/work", "dry-run": "true"})

	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/work", ".strategist"), fake.gotDir)
	assert.Zero(t, fake.applyCalls)
	assert.Contains(t, out, "managed (no change):     1")
	assert.Contains(t, out, "missing (will write): 1")
	assert.Contains(t, out, "auto_upgrade (will write): 1")
	assert.Contains(t, out, "customized (preserved): 1")
	assert.Contains(t, out, "orphaned (not deleted — review manually): 1")
	assert.Contains(t, out, "(dry run — nothing written)")
}

func TestUpgrade_ForceAppliesAndReportsBackup(t *testing.T) {
	fake := &fakeUpgrader{plan: samplePlan(), backupDir: "/work/.strategist/.upgrade-backups/20260923"}

	out, err := upgradeOutput(t, fakeUpgradeDeps(fake, ""), map[string]string{"target": "/work", "force": "true"})

	require.NoError(t, err)
	assert.Equal(t, 1, fake.applyCalls)
	assert.True(t, fake.gotForce)
	assert.Contains(t, out, "customized (will OVERWRITE — --force): 1")
	assert.Contains(t, out, "Backed up overwritten files to /work/.strategist/.upgrade-backups/20260923")
	assert.Contains(t, out, "Upgrade complete.")
}

func TestUpgrade_NoBackupDirPrintsOnlyCompletion(t *testing.T) {
	fake := &fakeUpgrader{plan: samplePlan()}

	out, err := upgradeOutput(t, fakeUpgradeDeps(fake, ""), map[string]string{"target": "/work"})

	require.NoError(t, err)
	assert.NotContains(t, out, "Backed up")
	assert.Contains(t, out, "Upgrade complete.")
}

func TestUpgrade_AllowDowngradeReachesTheServiceFactory(t *testing.T) {
	fake := &fakeUpgrader{}

	_, err := upgradeOutput(t, fakeUpgradeDeps(fake, ""), map[string]string{"target": "/work", "allow-downgrade": "true", "dry-run": "true"})

	require.NoError(t, err)
	require.NotNil(t, fake.factoryDowngr)
	assert.True(t, *fake.factoryDowngr)
}

func TestUpgrade_PlanAndApplyErrorsAreWrapped(t *testing.T) {
	planFail := &fakeUpgrader{planErr: errors.New("boom")}
	_, err := upgradeOutput(t, fakeUpgradeDeps(planFail, ""), map[string]string{"target": "/work"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade: boom")

	applyFail := &fakeUpgrader{applyErr: errors.New("disk full")}
	_, err = upgradeOutput(t, fakeUpgradeDeps(applyFail, ""), map[string]string{"target": "/work"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade: disk full")
}

func TestUpgrade_ResolveTargetErrorPropagates(t *testing.T) {
	deps := UpgradeDependencies{
		ResolveTarget:  func(string, bool) (string, error) { return "", errors.New("no home") },
		ServiceFactory: func(bool) UpgradeService { return &fakeUpgrader{} },
	}

	_, err := upgradeOutput(t, deps, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no home")
}

func TestUpgrade_MissingDependenciesFailClosed(t *testing.T) {
	_, err := upgradeOutput(t, UpgradeDependencies{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target resolver is not configured")

	deps := UpgradeDependencies{ResolveTarget: func(string, bool) (string, error) { return "/work", nil }}
	_, err = upgradeOutput(t, deps, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service factory is not configured")
}

func TestUpgrade_RollbackLatestWithNoBackupsFails(t *testing.T) {
	fake := &fakeUpgrader{}
	dir := t.TempDir()

	_, err := upgradeOutput(t, fakeUpgradeDeps(fake, ""), map[string]string{"target": dir, "rollback": "latest"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backups found")
	assert.Zero(t, fake.planCalls, "rollback must not plan an upgrade")
}

func TestUpgrade_RollbackUnknownStampFails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

	_, err := upgradeOutput(t, fakeUpgradeDeps(&fakeUpgrader{}, ""), map[string]string{"target": dir, "rollback": "19990101"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade rollback")
}

func TestRegister_AttachesInstallAndUpgrade(t *testing.T) {
	root := &cobra.Command{Use: "strategist"}

	Register(root, fakeDeps(&fakeInstaller{}), fakeUpgradeDeps(&fakeUpgrader{}, ""))

	names := map[string]bool{}
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	assert.True(t, names["install"])
	assert.True(t, names["upgrade"])
}

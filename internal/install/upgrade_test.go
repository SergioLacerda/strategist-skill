package install

import (
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUpgradeSource is a minimal domain.FileExtractor + domain.FileLister
// backed by an in-memory file map, standing in for the embedded defaults FS
// in upgrade tests.
type fakeUpgradeSource struct {
	files map[string][]byte
}

func (f fakeUpgradeSource) ReadFile(rel string) ([]byte, error) {
	data, ok := f.files[rel]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (f fakeUpgradeSource) Extract(string, bool) error { return nil } // unused by upgrade tests

func (f fakeUpgradeSource) AllPaths() ([]string, error) {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func upgradeTestService(files map[string][]byte) Service {
	src := fakeUpgradeSource{files: files}
	return Service{Extractor: src, Lister: src, Version: "1.0.0-test"}
}

func TestPlanUpgrade_FreshRoot_EverythingMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("a"), "b.yaml": []byte("b")})

	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	require.Len(t, plan.Entries, 2)
	for _, e := range plan.Entries {
		assert.Equal(t, domain.UpgradeMissing, e.State)
	}
}

func TestPlanUpgrade_HashEmbeddedPathsErrorPropagates(t *testing.T) {
	t.Parallel()

	svc := Service{Extractor: alwaysErrExtractor{}, Lister: alwaysAllPathsLister{[]string{"a.yaml"}}}

	_, err := svc.PlanUpgrade(t.TempDir())
	require.Error(t, err)
	require.ErrorContains(t, err, "upgrade: read embedded")
}

func TestPlanUpgrade_LoadInstallManifestErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, domain.InstallManifestRelPath), []byte("not json"), 0o644))
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1")})

	_, err := svc.PlanUpgrade(dir)
	require.Error(t, err)
	require.ErrorContains(t, err, "install: parse manifest")
}

func TestPlanUpgrade_ResolvePathErrorPropagates(t *testing.T) {
	t.Parallel()

	svc := upgradeTestService(map[string][]byte{"../escape.yaml": []byte("v1")})
	_, err := svc.PlanUpgrade(t.TempDir())
	require.Error(t, err)
	require.ErrorContains(t, err, "upgrade: resolve")
}

func TestPlanUpgrade_ReadCurrentHashErrorPropagates(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	t.Parallel()

	dir := t.TempDir()
	locked := filepath.Join(dir, "a.yaml")
	require.NoError(t, os.WriteFile(locked, []byte("x"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1")})
	_, err := svc.PlanUpgrade(dir)
	require.Error(t, err)
	require.ErrorContains(t, err, "upgrade: read a.yaml")
}

// alwaysAllPathsLister returns a fixed path list regardless of the backing extractor.
type alwaysAllPathsLister struct{ paths []string }

func (l alwaysAllPathsLister) AllPaths() ([]string, error) { return l.paths, nil }

func TestPlanUpgrade_RequiresLister(t *testing.T) {
	t.Parallel()

	svc := Service{Extractor: fakeUpgradeSource{}} // no Lister
	_, err := svc.PlanUpgrade(t.TempDir())
	require.Error(t, err)
}

func TestApplyUpgrade_WritesMissingAndManagedIsNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("content-a")})

	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	backupDir, err := svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)
	assert.Empty(t, backupDir, "missing files are written without a prior version to back up")

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "content-a", string(got))

	// A second plan against the now-current tree finds everything managed.
	plan2, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	require.Len(t, plan2.Entries, 1)
	assert.Equal(t, domain.UpgradeManaged, plan2.Entries[0].State)
}

func TestApplyUpgrade_PreservesCustomizedFileByDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})

	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	// User edits the installed file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("user-edited"), 0o644))

	plan2, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	require.Len(t, plan2.Entries, 1)
	assert.Equal(t, domain.UpgradeCustomized, plan2.Entries[0].State)

	backupDir, err := svc.ApplyUpgrade(dir, plan2, false)
	require.NoError(t, err)
	assert.Empty(t, backupDir, "nothing was overwritten without --force")

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "user-edited", string(got), "customized file must survive a non-force upgrade")
}

func TestApplyUpgrade_ForceOverwritesAndBacksUp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("user-edited"), 0o644))

	plan2, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	backupDir, err := svc.ApplyUpgrade(dir, plan2, true)
	require.NoError(t, err)
	require.NotEmpty(t, backupDir)

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "embedded-v1", string(got), "--force overwrites the customized file")

	backedUp, err := os.ReadFile(filepath.Join(backupDir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "user-edited", string(backedUp), "the pre-overwrite content must be recoverable from the backup")
}

func TestApplyUpgrade_AutoUpgradesUnmodifiedFileWhenEmbeddedChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	// Distribution ships a new version of a.yaml; the user never touched theirs.
	svc2 := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v2")})
	plan2, err := svc2.PlanUpgrade(dir)
	require.NoError(t, err)
	require.Len(t, plan2.Entries, 1)
	assert.Equal(t, domain.UpgradeAutoUpgrade, plan2.Entries[0].State)

	backupDir, err := svc2.ApplyUpgrade(dir, plan2, false)
	require.NoError(t, err)
	assert.NotEmpty(t, backupDir, "auto_upgrade still backs up the previous content")

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "embedded-v2", string(got))
}

func TestPlanUpgrade_DetectsOrphan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1"), "b.yaml": []byte("v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)

	// b.yaml is removed from the distribution.
	svc2 := upgradeTestService(map[string][]byte{"a.yaml": []byte("v1")})
	plan2, err := svc2.PlanUpgrade(dir)
	require.NoError(t, err)

	var sawOrphan bool
	for _, e := range plan2.Entries {
		if e.Path == "b.yaml" {
			sawOrphan = true
			assert.Equal(t, domain.UpgradeOrphaned, e.State)
		}
	}
	assert.True(t, sawOrphan, "b.yaml must be reported as orphaned")

	_, err = svc2.ApplyUpgrade(dir, plan2, true)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "b.yaml"), "orphaned files are never deleted automatically, even with --force")
}

func TestRollbackUpgrade_RestoresBackedUpFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"a.yaml": []byte("embedded-v1")})
	plan, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	_, err = svc.ApplyUpgrade(dir, plan, false)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("user-edited"), 0o644))

	plan2, err := svc.PlanUpgrade(dir)
	require.NoError(t, err)
	backupDir, err := svc.ApplyUpgrade(dir, plan2, true)
	require.NoError(t, err)
	stamp := filepath.Base(backupDir)

	stamps, err := ListUpgradeBackups(dir)
	require.NoError(t, err)
	require.Equal(t, []string{stamp}, stamps)

	count, err := RollbackUpgrade(dir, stamp)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	got, err := os.ReadFile(filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "user-edited", string(got), "rollback must restore the pre-upgrade content")
}

func TestRollbackUpgrade_UnknownStamp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := RollbackUpgrade(dir, "does-not-exist")
	require.Error(t, err)
}

func TestListUpgradeBackups_EmptyWhenNoneTaken(t *testing.T) {
	t.Parallel()

	stamps, err := ListUpgradeBackups(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, stamps)
}

func TestListUpgradeBackups_ResolveErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := ListUpgradeBackups("")
	require.Error(t, err)
	require.ErrorContains(t, err, "resolve backup dir")
}

func TestListUpgradeBackups_ReadDirErrorPropagates(t *testing.T) {
	testutil.SkipOnWindowsReadDirOfFile(t)
	t.Parallel()

	dir := t.TempDir()
	// upgradeBackupRelDir exists as a plain file, so os.ReadDir fails with
	// something other than "not exist" (which is otherwise tolerated as
	// "no backups yet").
	require.NoError(t, os.WriteFile(filepath.Join(dir, upgradeBackupRelDir), []byte("x"), 0o644))

	_, err := ListUpgradeBackups(dir)
	require.Error(t, err)
	require.ErrorContains(t, err, "list backups")
}

func TestRollbackUpgrade_ResolveErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := RollbackUpgrade("", "some-stamp")
	require.Error(t, err)
	require.ErrorContains(t, err, "resolve backup")
}

func TestRollbackUpgrade_ReadFileErrorPropagates(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	t.Parallel()

	dir := t.TempDir()
	backupDir := filepath.Join(dir, upgradeBackupRelDir, "20260101T000000Z")
	require.NoError(t, os.MkdirAll(backupDir, 0o755))
	locked := filepath.Join(backupDir, "a.yaml")
	require.NoError(t, os.WriteFile(locked, []byte("x"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	_, err := RollbackUpgrade(dir, "20260101T000000Z")
	require.Error(t, err)
	require.ErrorContains(t, err, "read backup")
}

func TestRollbackUpgrade_WriteErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	backupDir := filepath.Join(dir, upgradeBackupRelDir, "20260101T000000Z")
	require.NoError(t, os.MkdirAll(filepath.Join(backupDir, "blocked"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "blocked", "a.yaml"), []byte("x"), 0o644))
	// "blocked" exists at the restore target as a plain file, so the restore
	// target's parent directory cannot be created.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "blocked"), []byte("x"), 0o644))

	_, err := RollbackUpgrade(dir, "20260101T000000Z")
	require.Error(t, err)
	require.ErrorContains(t, err, "restore blocked")
}

func TestWriteUpgradeFile_ReadEmbeddedErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{}) // "missing.yaml" not in the embedded set
	err := svc.writeUpgradeFile(dir, "missing.yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "read embedded")
}

func TestWriteUpgradeFile_ResolveErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"../escape.yaml": []byte("x")})
	err := svc.writeUpgradeFile(dir, "../escape.yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "resolve")
}

func TestWriteUpgradeFile_WriteErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// "blocked" exists as a plain file, so writeUpgradeFile's target
	// (blocked/a.yaml) cannot have its parent directory created.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "blocked"), []byte("x"), 0o644))

	svc := upgradeTestService(map[string][]byte{"blocked/a.yaml": []byte("x")})
	err := svc.writeUpgradeFile(dir, "blocked/a.yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "write blocked/a.yaml")
}

func TestApplyUpgrade_WriteFileErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{})
	plan := UpgradePlan{Entries: []UpgradePlanEntry{{Path: "missing.yaml", State: domain.UpgradeMissing}}}

	_, err := svc.ApplyUpgrade(dir, plan, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "read embedded")
}

func TestApplyUpgrade_SnapshotErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := upgradeTestService(map[string][]byte{"../escape.yaml": []byte("x")})
	plan := UpgradePlan{Entries: []UpgradePlanEntry{{Path: "../escape.yaml", State: domain.UpgradeAutoUpgrade}}}

	_, err := svc.ApplyUpgrade(dir, plan, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "snapshot before write")
}

func TestApplyUpgrade_SaveManifestErrorPropagates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// The manifest target already exists as a directory, so the final
	// atomicWriteFile rename in saveInstallManifest fails.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, domain.InstallManifestRelPath), 0o755))

	svc := upgradeTestService(map[string][]byte{})
	_, err := svc.ApplyUpgrade(dir, UpgradePlan{}, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "save manifest")
}

func TestSnapshotBeforeUpgrade(t *testing.T) {
	t.Parallel()
	var svc Service

	t.Run("resolve backup dir error", func(t *testing.T) {
		t.Parallel()
		_, err := svc.snapshotBeforeUpgrade("", []string{"a.yaml"})
		require.Error(t, err)
		require.ErrorContains(t, err, "resolve backup dir")
	})

	t.Run("resolve src path error", func(t *testing.T) {
		t.Parallel()
		_, err := svc.snapshotBeforeUpgrade(t.TempDir(), []string{"../escape.yaml"})
		require.Error(t, err)
		require.ErrorContains(t, err, "resolve ../escape.yaml")
	})

	t.Run("missing file is skipped, not an error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		backupDir, err := svc.snapshotBeforeUpgrade(dir, []string{"never-existed.yaml"})
		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(backupDir, "never-existed.yaml"))
	})

	t.Run("unreadable file propagates a read error", func(t *testing.T) {
		skipIfPermissionTestUnsupported(t)
		t.Parallel()
		dir := t.TempDir()
		target := filepath.Join(dir, "locked.yaml")
		require.NoError(t, os.WriteFile(target, []byte("x"), 0o000))
		t.Cleanup(func() { _ = os.Chmod(target, 0o644) })

		_, err := svc.snapshotBeforeUpgrade(dir, []string{"locked.yaml"})
		require.Error(t, err)
		require.ErrorContains(t, err, "snapshot read locked.yaml")
	})

	t.Run("copies file content into a fresh timestamped backup dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("content"), 0o644))

		backupDir, err := svc.snapshotBeforeUpgrade(dir, []string{"a.yaml"})
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(backupDir, "a.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "content", string(got))
	})
}

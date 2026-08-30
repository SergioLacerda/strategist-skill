package install

import (
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

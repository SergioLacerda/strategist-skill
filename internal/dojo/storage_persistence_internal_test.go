package dojo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectStorage_ReportsMalformedHistory(t *testing.T) {
	base := t.TempDir()
	paths, err := NewStoragePaths(base, "history-scenario")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(paths.ScenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(paths.ScenarioDir, "criteria.yaml"), []byte("scenario: history-scenario\n"), 0o644))
	require.NoError(t, os.WriteFile(paths.HistoryPath, []byte("not-json\n"), 0o644))

	report, err := InspectStorage(base, "history-scenario")

	require.NoError(t, err)
	assert.True(t, report.HistoryPresent)
	assert.False(t, report.HistoryValid)
	assert.True(t, report.RecoveryRequired)
}

func TestInspectStorage_EmptyHistoryIsValid(t *testing.T) {
	base := t.TempDir()
	paths, err := NewStoragePaths(base, "empty-history")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(paths.ScenarioDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(paths.ScenarioDir, "criteria.yaml"), []byte("scenario: empty-history\n"), 0o644))
	require.NoError(t, os.WriteFile(paths.HistoryPath, nil, 0o644))

	report, err := InspectStorage(base, "empty-history")

	require.NoError(t, err)
	assert.True(t, report.HistoryValid)
	assert.False(t, report.RecoveryRequired)
}

func TestWriteLesson_RejectsInvalidStorage(t *testing.T) {
	err := WriteLesson("", failingResult("sample"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base path is empty")
}

func TestWriteLesson_ReportsStorageRootFailure(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(base, "dojo"), []byte("not-a-directory"), 0o644))

	err := WriteLesson(base, failingResult("sample"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create")
}

func TestAtomicWriteFile_ReportsCreateAndRenameFailures(t *testing.T) {
	base := t.TempDir()
	err := atomicWriteFile(filepath.Join(base, "missing", "result.json"), []byte("x"), 0o644)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create temporary file")

	target := filepath.Join(base, "target")
	require.NoError(t, os.Mkdir(target, 0o755))
	err = atomicWriteFile(target, []byte("x"), 0o644)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rename temporary file")
}

func TestStorageLock_RecoversStaleLock(t *testing.T) {
	base := t.TempDir()
	paths, err := NewStoragePaths(base, "lock-scenario")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(paths.DojoRoot, 0o755))
	lockPath := filepath.Join(paths.DojoRoot, storageLockName)
	require.NoError(t, os.Mkdir(lockPath, 0o700))
	old := time.Now().Add(-storageLockStale - time.Second)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	called := false
	err = withStorageLock(paths, func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
	assert.NoDirExists(t, lockPath)
}

func TestWriteHistoryRecord_ReportsClosedFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "history-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = writeHistoryRecord(f, []byte("{}\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write history record")
}

func TestStorageWritersReportDirectoryFailures(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	paths := StoragePaths{
		LastRunDir: filepath.Join(blocker, "last-run"),
		ResultPath: filepath.Join(blocker, "last-run", "result.json"),
	}

	err := writeResultJSON(paths, ResultRecord{Scenario: "sample"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create")
}

func failingResult(scenario string) domain.DojoCheckResult {
	return domain.DojoCheckResult{Scenario: scenario, Items: []domain.DojoCheckItem{{Passed: false}}}
}

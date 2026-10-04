package mission

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSaveAtomicWriteLeavesPreviousState(t *testing.T) {
	root := t.TempDir()
	missionID := "m-atomic"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	initial := domain.MissionEngineStatus{MissionID: missionID, Phase: "BOOTSTRAP", State: "INIT"}
	require.NoError(t, Save(root, initial))

	dir := filepath.Join(root, "missions")
	target := Path(root, missionID)
	stray, err := os.CreateTemp(dir, filepath.Base(target)+".tmp-*")
	require.NoError(t, err)
	_, err = stray.WriteString("{not valid json")
	require.NoError(t, err)
	require.NoError(t, stray.Close())

	_, loaded, err := Load(root, missionID)
	require.NoError(t, err)
	require.Equal(t, initial.Phase, loaded.Phase)
	require.Equal(t, initial.State, loaded.State)
	require.NoError(t, os.Remove(stray.Name()))

	next := domain.MissionEngineStatus{MissionID: missionID, Phase: "INTAKE", State: "INIT"}
	require.NoError(t, Save(root, next))
	_, loaded, err = Load(root, missionID)
	require.NoError(t, err)
	require.Equal(t, next.Phase, loaded.Phase)
}

func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	root := t.TempDir()
	status := domain.MissionEngineStatus{MissionID: "m-clean", Phase: "BOOTSTRAP", State: "INIT"}
	require.NoError(t, Save(root, status))
	entries, err := os.ReadDir(filepath.Join(root, "missions"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "m-clean.json", entries[0].Name())
}

func TestLockSerializesConcurrentCallers(t *testing.T) {
	root := t.TempDir()
	const workers, incrementsPerWorker = 12, 25
	var counter int64
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		startLockWorker(&wg, root, incrementsPerWorker, &counter, errs)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int64(workers*incrementsPerWorker), atomic.LoadInt64(&counter))
}

func startLockWorker(wg *sync.WaitGroup, root string, increments int, counter *int64, errs chan<- error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < increments; j++ {
			err := Lock(root, "m-lock", func() error {
				current := atomic.LoadInt64(counter)
				atomic.StoreInt64(counter, current+1)
				return nil
			})
			if err != nil {
				errs <- err
				return
			}
		}
	}()
}

func TestLockScopesDifferentMissionsIndependently(t *testing.T) {
	root := t.TempDir()
	done := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = Lock(root, "m-a", func() error {
			close(done)
			<-release
			return nil
		})
	}()
	<-done
	require.NoError(t, Lock(root, "m-b", func() error { return nil }))
	close(release)
}

func TestPersistenceValidationAndRestoreErrors(t *testing.T) {
	require.Error(t, ValidateMissionID(""))
	require.Error(t, ValidateMissionID("../escape"))
	require.NoError(t, ValidateMissionID("20261003-valid-mission"))

	root := t.TempDir()
	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	require.ErrorContains(t, Save(file, domain.MissionEngineStatus{MissionID: "m"}), "create mission directory")
	blocked := t.TempDir()
	require.NoError(t, os.MkdirAll(Path(blocked, "m"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(Path(blocked, "m"), "child"), nil, 0o644))
	require.ErrorContains(t, Save(blocked, domain.MissionEngineStatus{MissionID: "m"}), "write mission state")

	require.NoError(t, RequireNoExisting(root, "fresh"))
	require.NoError(t, Save(root, domain.MissionEngineStatus{MissionID: "taken", Phase: domain.PhaseBootstrap, State: domain.StateInit}))
	require.ErrorContains(t, RequireNoExisting(root, "taken"), "already exists")
	_, _, err := Load(root, "ghost")
	require.ErrorContains(t, err, "not found")

	bad := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(bad, "missions"), 0o755))
	require.NoError(t, os.WriteFile(Path(bad, "m"), []byte("{nope"), 0o644))
	_, _, err = Load(bad, "m")
	require.ErrorContains(t, err, "invalid persisted state")
	require.NoError(t, os.WriteFile(Path(bad, "m"), []byte(`{"mission_id":"m","phase":"NOPE","state":"NOPE"}`), 0o644))
	_, _, err = Load(bad, "m")
	require.ErrorContains(t, err, "restore mission state")
}

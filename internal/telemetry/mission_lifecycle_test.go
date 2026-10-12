package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lifecycleRecord(id, kind, phase, run, reason string) MissionLifecycleRecord {
	return MissionLifecycleRecord{
		SchemaVersion: MissionLifecycleSchemaVersion, MarkerID: id, MarkerKind: kind,
		MissionID: "m-1", Phase: phase, PhaseRunID: run, EndReason: reason,
		ObservedAt: "2026-10-10T10:00:00Z",
	}
}

func TestValidateMissionLifecycleRecord(t *testing.T) {
	valid := lifecycleRecord("m-1|DISCOVERY|r1|phase_started", "phase_started", "DISCOVERY", "r1", "")
	require.NoError(t, ValidateMissionLifecycleRecord(valid))

	ended := lifecycleRecord("m-1|DISCOVERY|r1|phase_ended", "phase_ended", "DISCOVERY", "r1", "completed")
	require.NoError(t, ValidateMissionLifecycleRecord(ended))

	missionEnd := lifecycleRecord("m-1|DONE|-|mission_ended|completed", "mission_ended", "DONE", "", "completed")
	require.NoError(t, ValidateMissionLifecycleRecord(missionEnd))

	cases := map[string]func(*MissionLifecycleRecord){
		"schema version":     func(r *MissionLifecycleRecord) { r.SchemaVersion = "2" },
		"marker id":          func(r *MissionLifecycleRecord) { r.MarkerID = "" },
		"marker kind":        func(r *MissionLifecycleRecord) { r.MarkerKind = "phase_paused" },
		"mission id":         func(r *MissionLifecycleRecord) { r.MissionID = "" },
		"phase":              func(r *MissionLifecycleRecord) { r.Phase = "DELIVERY" },
		"run id on phase":    func(r *MissionLifecycleRecord) { r.PhaseRunID = "" },
		"observed at":        func(r *MissionLifecycleRecord) { r.ObservedAt = "yesterday" },
		"end reason missing": func(r *MissionLifecycleRecord) { r.MarkerKind = "phase_ended"; r.EndReason = "" },
		"end reason unknown": func(r *MissionLifecycleRecord) { r.MarkerKind = "phase_ended"; r.EndReason = "gave_up" },
		"end reason on start": func(r *MissionLifecycleRecord) {
			r.EndReason = "completed"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			rec := valid
			mutate(&rec)
			require.Error(t, ValidateMissionLifecycleRecord(rec))
		})
	}
}

func TestAppendMissionLifecycleRecordsIsIdempotentByMarkerID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "mission-lifecycle.jsonl")
	start := lifecycleRecord("m-1|BOOTSTRAP|r1|phase_started", "phase_started", "BOOTSTRAP", "r1", "")

	n, err := AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{start})
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{start})
	require.NoError(t, err)
	assert.Equal(t, 0, n, "a replayed marker is a no-op")

	got, err := ReadMissionLifecycle(path)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestAppendMissionLifecycleRecordsRejectsTheWholeBatchWhenOneIsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission-lifecycle.jsonl")
	good := lifecycleRecord("a", "phase_started", "INTAKE", "r1", "")
	bad := lifecycleRecord("b", "phase_started", "NOPE", "r1", "")

	_, err := AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{good, bad})
	require.Error(t, err)

	got, readErr := ReadMissionLifecycle(path)
	require.NoError(t, readErr)
	assert.Empty(t, got)
}

func TestAppendMissionLifecycleRecordsReportsAnUnwritablePath(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "memory")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	_, err := AppendMissionLifecycleRecords(filepath.Join(blocker, "mission-lifecycle.jsonl"),
		[]MissionLifecycleRecord{lifecycleRecord("a", "phase_started", "INTAKE", "r1", "")})
	require.Error(t, err)
}

func TestReadMissionLifecycleToleratesMissingFileAndMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission-lifecycle.jsonl")
	got, err := ReadMissionLifecycle(path)
	require.NoError(t, err)
	assert.Nil(t, got)

	good := lifecycleRecord("a", "phase_started", "INTAKE", "r1", "")
	_, err = AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{good})
	require.NoError(t, err)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("{not json}\n\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got, err = ReadMissionLifecycle(path)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestReadMissionLifecycleForFiltersByMission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission-lifecycle.jsonl")
	a := lifecycleRecord("a", "phase_started", "INTAKE", "r1", "")
	b := lifecycleRecord("b", "phase_started", "INTAKE", "r1", "")
	b.MissionID = "m-2"
	_, err := AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{a, b})
	require.NoError(t, err)

	got, err := ReadMissionLifecycleFor(path, "m-2")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "b", got[0].MarkerID)
}

func TestResolveOpenPhaseRunRequiresOneOpenRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
	recs := []MissionLifecycleRecord{
		lifecycleRecord("m-1", string(domain.MarkerPhaseStarted), string(domain.PhaseRefinement), "r1", ""),
	}
	_, err := AppendMissionLifecycleRecords(path, recs)
	require.NoError(t, err)
	got, err := ResolveOpenPhaseRun(path, "m-1", string(domain.PhaseRefinement))
	require.NoError(t, err)
	assert.Equal(t, "r1", got)

	_, err = AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{lifecycleRecord("m-1|REFINEMENT|r1|phase_ended", string(domain.MarkerPhaseEnded), string(domain.PhaseRefinement), "r1", string(domain.EndCompleted))})
	require.NoError(t, err)
	_, err = ResolveOpenPhaseRun(path, "m-1", string(domain.PhaseRefinement))
	assert.Error(t, err)
}

func appendWorker(path string, w, count int) error {
	for i := 0; i < count; i++ {
		rec := lifecycleRecord(fmt.Sprintf("w%d-%d", w, i), "phase_started", "INTAKE", "r1", "")
		if _, err := AppendMissionLifecycleRecords(path, []MissionLifecycleRecord{rec}); err != nil {
			return err
		}
	}
	return nil
}

func TestAppendMissionLifecycleRecordsConcurrentAppendersLoseNoLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the file lock is a documented no-op on Windows")
	}
	path := filepath.Join(t.TempDir(), "mission-lifecycle.jsonl")
	const writers, perWriter = 8, 10
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- appendWorker(path, w, perWriter)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	got, err := ReadMissionLifecycle(path)
	require.NoError(t, err)
	assert.Len(t, got, writers*perWriter)
}

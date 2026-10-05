package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func usageInt(v int64) *int64 { return &v }

func usageRecord(id string) MissionUsageRecord {
	return MissionUsageRecord{
		SchemaVersion: MissionUsageSchemaVersion, RecordID: id, MissionID: "m-1",
		Phase: "DISCOVERY", Role: "ranger", Attempt: 1, UsageScope: "invocation",
		RecordKind: "incremental", TokensInput: usageInt(1200), TokensOutput: usageInt(300),
		Provider: "unknown", Model: "unknown",
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified",
		ObservedAt: "2026-10-10T10:00:00Z",
	}
}

func TestValidateMissionUsageRecord(t *testing.T) {
	valid := usageRecord("r-1")
	require.NoError(t, ValidateMissionUsageRecord(valid))

	unattributed := valid
	unattributed.Phase = "UNATTRIBUTED"
	require.NoError(t, ValidateMissionUsageRecord(unattributed))

	onlyOutput := valid
	onlyOutput.TokensInput = nil
	require.NoError(t, ValidateMissionUsageRecord(onlyOutput), "one observed member is enough")

	snapshot := valid
	snapshot.RecordKind = "cumulative_snapshot"
	snapshot.SessionID = "sess-1"
	require.NoError(t, ValidateMissionUsageRecord(snapshot))

	cases := map[string]func(*MissionUsageRecord){
		"schema":                  func(r *MissionUsageRecord) { r.SchemaVersion = "9" },
		"record id":               func(r *MissionUsageRecord) { r.RecordID = "" },
		"mission":                 func(r *MissionUsageRecord) { r.MissionID = "" },
		"phase gate":              func(r *MissionUsageRecord) { r.Phase = "APPROVAL_GATE" },
		"phase unknown":           func(r *MissionUsageRecord) { r.Phase = "DELIVERY" },
		"role":                    func(r *MissionUsageRecord) { r.Role = "" },
		"attempt":                 func(r *MissionUsageRecord) { r.Attempt = 0 },
		"scope":                   func(r *MissionUsageRecord) { r.UsageScope = "" },
		"kind":                    func(r *MissionUsageRecord) { r.RecordKind = "running_total" },
		"no tokens":               func(r *MissionUsageRecord) { r.TokensInput, r.TokensOutput = nil, nil },
		"negative input":          func(r *MissionUsageRecord) { r.TokensInput = usageInt(-1) },
		"negative cache":          func(r *MissionUsageRecord) { r.TokensCache = usageInt(-5) },
		"source":                  func(r *MissionUsageRecord) { r.MeasurementSource = "oracle" },
		"quality":                 func(r *MissionUsageRecord) { r.MeasurementQuality = "perfect" },
		"measured claim by agent": func(r *MissionUsageRecord) { r.MeasurementQuality = "provider_measured" },
		"observed at":             func(r *MissionUsageRecord) { r.ObservedAt = "tomorrow" },
		"snapshot without session": func(r *MissionUsageRecord) {
			r.RecordKind = "cumulative_snapshot"
			r.SessionID = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			rec := valid
			mutate(&rec)
			require.Error(t, ValidateMissionUsageRecord(rec))
		})
	}
}

func TestMissionUsageRecordIDKeys(t *testing.T) {
	withEvent := usageRecord("")
	withEvent.EventID = "evt-1"
	assert.Equal(t, "event|m-1|evt-1", MissionUsageRecordID(withEvent))

	withInvocation := usageRecord("")
	withInvocation.InvocationID = "inv_abc"
	withInvocation.Attempt = 2
	assert.Equal(t, "inv|m-1|inv_abc|2|invocation", MissionUsageRecordID(withInvocation))

	snap := withInvocation
	snap.RecordKind = "cumulative_snapshot"
	snap.SessionID = "s1"
	assert.Contains(t, MissionUsageRecordID(snap), "|snap|s1|1200|300")

	unkeyed := usageRecord("")
	a, b := MissionUsageRecordID(unkeyed), MissionUsageRecordID(unkeyed)
	assert.Equal(t, a, b, "deterministic for the same record")
	unkeyed.ObservedAt = "2026-10-10T10:00:01Z"
	assert.NotEqual(t, a, MissionUsageRecordID(unkeyed), "an unkeyed record is never merged with another observation")
}

func TestAppendMissionUsageRecordIsIdempotentAndDetectsConflicts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "mission-usage.jsonl")
	rec := usageRecord("inv|m-1|inv_abc|1|invocation")

	appended, err := AppendMissionUsageRecord(path, rec)
	require.NoError(t, err)
	assert.True(t, appended)

	appended, err = AppendMissionUsageRecord(path, rec)
	require.NoError(t, err)
	assert.False(t, appended, "same key and same counts is a duplicate")

	conflict := rec
	conflict.TokensInput = usageInt(9999)
	_, err = AppendMissionUsageRecord(path, conflict)
	require.ErrorIs(t, err, ErrMissionUsageConflict)

	got, err := ReadMissionUsageFor(path, "m-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(1200), *got[0].TokensInput)
}

func TestAppendMissionUsageRecordValidatesAndReportsUnwritablePaths(t *testing.T) {
	dir := t.TempDir()
	bad := usageRecord("x")
	bad.Phase = "NOPE"
	_, err := AppendMissionUsageRecord(filepath.Join(dir, "u.jsonl"), bad)
	require.Error(t, err)

	blocker := filepath.Join(dir, "memory")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	_, err = AppendMissionUsageRecord(filepath.Join(blocker, "u.jsonl"), usageRecord("x"))
	require.Error(t, err)
}

func TestReadMissionUsageToleratesMissingFileAndMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "u.jsonl")
	got, err := ReadMissionUsageFor(path, "m-1")
	require.NoError(t, err)
	assert.Nil(t, got)

	_, err = AppendMissionUsageRecord(path, usageRecord("a"))
	require.NoError(t, err)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("{oops\n\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got, err = ReadMissionUsageFor(path, "m-1")
	require.NoError(t, err)
	assert.Len(t, got, 1)

	other, err := ReadMissionUsageFor(path, "m-other")
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestAppendMissionUsageRecordConcurrentWritersLoseNoLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the file lock is a documented no-op on Windows")
	}
	path := filepath.Join(t.TempDir(), "u.jsonl")
	const writers, per = 8, 10
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- usageWorker(path, w, per)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	got, err := ReadMissionUsageFor(path, "m-1")
	require.NoError(t, err)
	assert.Len(t, got, writers*per)
}

func usageWorker(path string, w, count int) error {
	for i := 0; i < count; i++ {
		if _, err := AppendMissionUsageRecord(path, usageRecord(fmt.Sprintf("w%d-%d", w, i))); err != nil {
			return err
		}
	}
	return nil
}

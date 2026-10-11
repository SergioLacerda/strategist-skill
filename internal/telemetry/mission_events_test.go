package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func roundTripFile(t *testing.T, events ...Event) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.jsonl")
	for _, e := range events {
		require.NoError(t, AppendEventLine(path, e))
	}
	return path
}

func TestLifecycleEventRoundTripsThroughTheJSONLStream(t *testing.T) {
	recs := []MissionLifecycleRecord{
		{SchemaVersion: "1", MarkerID: "m-1|BOOTSTRAP|-|mission_started", MarkerKind: "mission_started", MissionID: "m-1", Phase: "BOOTSTRAP", Stage: "SHORT", ObservedAt: "2026-10-10T10:00:00Z"},
		{SchemaVersion: "1", MarkerID: "m-1|DISCOVERY|r1|phase_started", MarkerKind: "phase_started", MissionID: "m-1", Phase: "DISCOVERY", PhaseRunID: "r1", ObservedAt: "2026-10-10T10:02:00Z"},
		{SchemaVersion: "1", MarkerID: "m-1|DISCOVERY|r1|phase_ended", MarkerKind: "phase_ended", MissionID: "m-1", Phase: "DISCOVERY", PhaseRunID: "r1", EndReason: "completed", ObservedAt: "2026-10-10T10:24:00Z"},
		{SchemaVersion: "1", MarkerID: "m-1|DONE|-|mission_ended|completed", MarkerKind: "mission_ended", MissionID: "m-1", Phase: "DONE", EndReason: "completed", ObservedAt: "2026-10-10T11:25:00Z"},
	}
	var events []Event
	for _, r := range recs {
		e, err := NewMissionLifecycleEvent(r)
		require.NoError(t, err)
		events = append(events, e)
	}
	assert.Equal(t, "strategist.mission.started", events[0].Name)
	assert.Equal(t, "strategist.mission.phase.started", events[1].Name)
	assert.Equal(t, "strategist.mission.phase.ended", events[2].Name)
	assert.Equal(t, "strategist.mission.ended", events[3].Name)
	assert.Equal(t, "m-1", events[1].Attributes[AttrCorrelationID])
	assert.Equal(t, MissionLifecycleContractID, events[1].Attributes[AttrEventContractID])
	assert.Equal(t, "m-1", events[1].RunID)

	got, malformed, err := ReadEventLines(roundTripFile(t, events...))
	require.NoError(t, err)
	assert.Zero(t, malformed)
	require.Len(t, got, len(recs))
	for i, e := range got {
		back, ok := MissionLifecycleRecordFromEvent(e)
		require.True(t, ok)
		assert.Equal(t, recs[i], back)
	}
}

func TestUsageEventRoundTripKeepsUnobservedCountsNull(t *testing.T) {
	in, out := int64(1200), int64(0) // an observed zero is kept as zero
	rec := MissionUsageRecord{
		SchemaVersion: "1", RecordID: "inv|m-1|inv_1|1|invocation", MissionID: "m-1", Phase: "DISCOVERY", Role: "ranger",
		InvocationID: "inv_1", Attempt: 2, UsageScope: "invocation", RecordKind: "incremental",
		TokensInput: &in, TokensOutput: &out, Provider: "anthropic", Model: "m-a",
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", Calls: 3,
		ObservedAt: "2026-10-10T10:10:00Z",
	}
	e := NewMissionUsageEvent(rec)
	assert.Equal(t, "strategist.usage.recorded", e.Name)
	_, hasCache := e.Attributes[AttrUsageTokensCache]
	assert.False(t, hasCache, "an unobserved count is an absent attribute, never zero")

	got, _, err := ReadEventLines(roundTripFile(t, e))
	require.NoError(t, err)
	require.Len(t, got, 1)
	back, ok := MissionUsageRecordFromEvent(got[0])
	require.True(t, ok)
	assert.Equal(t, rec, back)
	assert.Nil(t, back.TokensCache)
	require.NotNil(t, back.TokensOutput)
	assert.Equal(t, int64(0), *back.TokensOutput)
}

func TestUsageEventNeverCarriesContentFields(t *testing.T) {
	in := int64(5)
	e := NewMissionUsageEvent(MissionUsageRecord{
		SchemaVersion: "1", RecordID: "r", MissionID: "m-1", Phase: "DISCOVERY", Role: "ranger", Attempt: 1,
		UsageScope: "invocation", RecordKind: "incremental", TokensInput: &in,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:10:00Z",
	})
	for key := range e.Attributes {
		assert.NotContains(t, key, "prompt")
		assert.NotContains(t, key, "content")
		assert.NotContains(t, key, "response")
	}
}

func TestReadEventLinesSkipsMalformedAndForeignEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	e, err := NewMissionLifecycleEvent(MissionLifecycleRecord{SchemaVersion: "1", MarkerID: "a", MarkerKind: "mission_started", MissionID: "m-1", Phase: "BOOTSTRAP", ObservedAt: "2026-10-10T10:00:00Z"})
	require.NoError(t, err)
	require.NoError(t, AppendEventLine(path, e))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("{broken\n\n" + `{"name":"strategist.something.else","timestamp":"2026-10-10T10:00:00.000000000Z","complete":true}` + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	got, malformed, err := ReadEventLines(path)
	require.NoError(t, err)
	assert.Equal(t, 1, malformed)
	require.Len(t, got, 2, "a foreign event is readable; it is just not a mission record")
	_, ok := MissionLifecycleRecordFromEvent(got[1])
	assert.False(t, ok)
	_, ok = MissionUsageRecordFromEvent(got[1])
	assert.False(t, ok)

	missing, malformed, err := ReadEventLines(filepath.Join(t.TempDir(), "none.jsonl"))
	require.NoError(t, err)
	assert.Nil(t, missing)
	assert.Zero(t, malformed)
}

func TestLifecycleEventRejectsAnInvalidRecord(t *testing.T) {
	_, err := NewMissionLifecycleEvent(MissionLifecycleRecord{MarkerKind: "phase_paused"})
	require.Error(t, err)
}

func TestCountUnreadableJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	require.NoError(t, os.WriteFile(path, []byte("{\"a\":1}\n\n{broken\n[1,2]\n\"text\"\n{\"b\":2}\n"), 0o600))
	n, err := CountUnreadableJSONLines(path)
	require.NoError(t, err)
	assert.Equal(t, 3, n, "broken json, an array and a bare string are not records")

	n, err = CountUnreadableJSONLines(filepath.Join(t.TempDir(), "none.jsonl"))
	require.NoError(t, err)
	assert.Zero(t, n)
}

package application_test

import (
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hostInput() application.HostObservedUsageInput {
	return application.HostObservedUsageInput{
		MissionID: "m-1", Phase: "discovery", Role: "ranger", Host: "claude", SessionID: "sess-1",
		Provider: "anthropic", Models: []string{"m-a"},
		WindowStart: time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC), WindowEnd: time.Date(2026, 10, 10, 11, 0, 0, 0, time.UTC),
		Calls: 7, Input: tokens(5000), Output: tokens(900), Cache: tokens(4000),
	}
}

func TestRecordHostObservedUsageIsHostMeasuredAndWindowKeyed(t *testing.T) {
	store := &usageStore{known: true}
	res, err := application.RecordHostObservedUsage("root", "base", hostInput(), usageNow, store.ports())
	require.NoError(t, err)

	rec := res.Record
	assert.Equal(t, "host_counter", rec.MeasurementSource)
	assert.Equal(t, "host_measured", rec.MeasurementQuality)
	assert.Equal(t, "window", rec.UsageScope)
	assert.Equal(t, "incremental", rec.RecordKind)
	assert.Equal(t, "host|claude|sess-1|2026-10-10T10:00:00Z|2026-10-10T11:00:00Z", rec.EventID)
	assert.False(t, rec.DedupUnavailable)
	assert.Equal(t, "DISCOVERY", rec.Phase)
	assert.Equal(t, "m-a", rec.Model)
	assert.Equal(t, 7, rec.Calls)
	assert.Equal(t, int64(4000), *rec.TokensCache)
}

func TestRecordHostObservedUsageRefusesToRecordNothingAsZero(t *testing.T) {
	store := &usageStore{known: true}
	in := hostInput()
	in.Calls, in.Input, in.Output = 0, nil, nil
	_, err := application.RecordHostObservedUsage("root", "base", in, usageNow, store.ports())
	require.ErrorContains(t, err, "nothing is recorded as zero")
	assert.Empty(t, store.records)
}

func TestRecordHostObservedUsageRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(*application.HostObservedUsageInput){
		"phase":        func(i *application.HostObservedUsageInput) { i.Phase = "approval_gate" },
		"role":         func(i *application.HostObservedUsageInput) { i.Role = "" },
		"host":         func(i *application.HostObservedUsageInput) { i.Host = "gemini" },
		"session":      func(i *application.HostObservedUsageInput) { i.SessionID = "" },
		"window":       func(i *application.HostObservedUsageInput) { i.WindowEnd = i.WindowStart },
		"mission":      func(i *application.HostObservedUsageInput) { i.MissionID = "" },
		"unknown mdls": func(i *application.HostObservedUsageInput) { i.Models = nil; i.Phase = "nope" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			store := &usageStore{known: true}
			in := hostInput()
			mutate(&in)
			_, err := application.RecordHostObservedUsage("root", "base", in, usageNow, store.ports())
			require.Error(t, err)
			assert.Empty(t, store.records)
		})
	}
}

func TestRecordHostObservedUsageKeepsTheArtifactRequirementAndLabelsMixedModels(t *testing.T) {
	_, err := application.RecordHostObservedUsage("root", "base", hostInput(), usageNow, (&usageStore{known: false}).ports())
	require.ErrorContains(t, err, "unknown mission_id")

	store := &usageStore{known: true}
	in := hostInput()
	in.Models = []string{"m-a", "m-b"}
	res, err := application.RecordHostObservedUsage("root", "base", in, usageNow, store.ports())
	require.NoError(t, err)
	assert.Equal(t, "mixed", res.Record.Model)

	_, err = application.RecordHostObservedUsage("root", "base", hostInput(), usageNow, application.MissionUsagePorts{})
	require.Error(t, err)
}

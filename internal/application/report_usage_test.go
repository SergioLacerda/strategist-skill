package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tokens(v int64) *int64 { return &v }

type usageStore struct {
	records []application.MissionUsageRecord
	known   bool
	err     error
}

func (s *usageStore) ports() application.MissionUsagePorts {
	return application.MissionUsagePorts{
		MissionKnown: func(string, string) bool { return s.known },
		Append: func(_ string, r application.MissionUsageRecord) (bool, error) {
			if s.err != nil {
				return false, s.err
			}
			s.records = append(s.records, r)
			return true, nil
		},
	}
}

var usageNow = time.Date(2026, time.October, 3, 1, 2, 3, 0, time.FixedZone("BRT", -3*60*60))

func baseUsage() application.MissionUsageInput {
	return application.MissionUsageInput{
		MissionID: "m-1", Phase: "discovery", Role: "ranger",
		TokensInput: tokens(12), TokensOutput: tokens(7),
	}
}

func TestRecordMissionUsageAppliesDocumentedDefaults(t *testing.T) {
	store := &usageStore{known: true}
	res, err := application.RecordMissionUsage("root", "base", baseUsage(), usageNow, store.ports())
	require.NoError(t, err)

	rec := res.Record
	assert.True(t, res.Appended)
	assert.Equal(t, "DISCOVERY", rec.Phase)
	assert.Equal(t, 1, rec.Attempt)
	assert.Equal(t, "invocation", rec.UsageScope)
	assert.Equal(t, "incremental", rec.RecordKind)
	assert.Equal(t, "agent_report", rec.MeasurementSource)
	assert.Equal(t, "reported_unverified", rec.MeasurementQuality)
	assert.Equal(t, "unknown", rec.Provider)
	assert.True(t, rec.DedupUnavailable, "no invocation id and no event id means no stable identity")
	assert.Equal(t, "2026-10-03T04:02:03Z", rec.ObservedAt)
	assert.Equal(t, int64(12), *rec.TokensInput)
}

func TestRecordMissionUsageKeyedRecordIsDedupable(t *testing.T) {
	store := &usageStore{known: true}
	in := baseUsage()
	in.InvocationID = "inv_abc"
	res, err := application.RecordMissionUsage("root", "base", in, usageNow, store.ports())
	require.NoError(t, err)
	assert.False(t, res.Record.DedupUnavailable)

	in2 := baseUsage()
	in2.EventID = "evt-1"
	res, err = application.RecordMissionUsage("root", "base", in2, usageNow, store.ports())
	require.NoError(t, err)
	assert.False(t, res.Record.DedupUnavailable)
}

func TestRecordMissionUsageRefusesAnUnknownMission(t *testing.T) {
	store := &usageStore{known: false}
	_, err := application.RecordMissionUsage("root", "base", baseUsage(), usageNow, store.ports())
	require.ErrorContains(t, err, `unknown mission_id "m-1"`)
	assert.Empty(t, store.records)
}

func TestRecordMissionUsageRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(*application.MissionUsageInput){
		"mission":              func(i *application.MissionUsageInput) { i.MissionID = "" },
		"phase":                func(i *application.MissionUsageInput) { i.Phase = "" },
		"gate has no tokens":   func(i *application.MissionUsageInput) { i.Phase = "approval_gate" },
		"unknown phase":        func(i *application.MissionUsageInput) { i.Phase = "someday" },
		"role":                 func(i *application.MissionUsageInput) { i.Role = "" },
		"no counts":            func(i *application.MissionUsageInput) { i.TokensInput, i.TokensOutput = nil, nil },
		"negative":             func(i *application.MissionUsageInput) { i.TokensOutput = tokens(-1) },
		"bad kind":             func(i *application.MissionUsageInput) { i.RecordKind = "running_total" },
		"snapshot no session":  func(i *application.MissionUsageInput) { i.RecordKind = "cumulative_snapshot" },
		"attempt":              func(i *application.MissionUsageInput) { i.Attempt = -2 },
		"unsupported source":   func(i *application.MissionUsageInput) { i.Source = "provider_response" },
		"measured by an agent": func(i *application.MissionUsageInput) { i.Quality = "provider_measured" },
		"stronger than source": func(i *application.MissionUsageInput) { i.Source = "estimator"; i.Quality = "reported_unverified" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			store := &usageStore{known: true}
			in := baseUsage()
			mutate(&in)
			_, err := application.RecordMissionUsage("root", "base", in, usageNow, store.ports())
			require.Error(t, err)
			assert.Empty(t, store.records)
		})
	}
}

func TestRecordMissionUsageEstimatorAndSnapshot(t *testing.T) {
	store := &usageStore{known: true}
	in := baseUsage()
	in.Source = "estimator"
	in.Quality = "tokenizer_estimated"
	in.RecordKind = "cumulative_snapshot"
	in.SessionID = "sess-9"
	in.Phase = "unattributed"
	res, err := application.RecordMissionUsage("root", "base", in, usageNow, store.ports())
	require.NoError(t, err)
	assert.Equal(t, "UNATTRIBUTED", res.Record.Phase)
	assert.Equal(t, "tokenizer_estimated", res.Record.MeasurementQuality)
	assert.Equal(t, "sess-9", res.Record.SessionID)
}

func TestRecordMissionUsagePropagatesPortFailures(t *testing.T) {
	wantErr := errors.New("append failed")
	store := &usageStore{known: true, err: wantErr}
	_, err := application.RecordMissionUsage("root", "base", baseUsage(), usageNow, store.ports())
	require.ErrorIs(t, err, wantErr)

	_, err = application.RecordMissionUsage("root", "base", baseUsage(), usageNow, application.MissionUsagePorts{})
	require.EqualError(t, err, "mission report-usage: usage ports are required")
}
